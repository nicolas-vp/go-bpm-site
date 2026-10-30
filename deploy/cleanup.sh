#!/bin/bash
# Очистка общего демо go-bpm: экземпляры, которые запускали посетители сайта. На сервере - /opt/go-bpm-site/cleanup.sh,
# запускается cron раз в сутки (/etc/cron.d/go-bpm-site-cleanup).
#
# Удаляются завершённые, упавшие и отменённые экземпляры старше 3 дней и экземпляры, которые больше 14 дней ждут задачу
# пользователя или сообщение, - вместе с историей, правилами DMN, задачами пользователей и подписками на сообщения.
# Упавшие задачи очереди старше 14 дней и просроченные сообщения без подписки - тоже (выполненные задачи очередь
# чистит сама).
set -euo pipefail

docker exec -i go-bpm-postgres-1 psql -U go_bpm -d go_bpm -v ON_ERROR_STOP=1 -q <<'SQL'
begin;
-- старые корневые экземпляры и все их потомки: подпроцессы и call activity удаляются вместе с родителем
create temporary table old_instance on commit drop as
with recursive old(instance_uid) as (
    select instance_uid from bpm_process_instance i
    where (status in ('completed', 'failed', 'cancelled') and updated_at < now() - interval '3 days')
       or (status = 'running' and updated_at < now() - interval '14 days'
           and (exists (select 1 from bpm_user_task t where t.instance_uid = i.instance_uid and t.status = 'created')
                or exists (select 1 from bpm_message_subscription m where m.instance_uid = i.instance_uid)))
    union
    select c.instance_uid from bpm_process_instance c join old o on c.parent_instance_uid = o.instance_uid
)
select instance_uid from old;
delete from bpm_history where instance_uid in (select instance_uid from old_instance);
delete from bpm_rules_history where instance_uid in (select instance_uid from old_instance);
delete from bpm_user_task where instance_uid in (select instance_uid from old_instance);
delete from bpm_message_subscription where instance_uid in (select instance_uid from old_instance);
delete from bpm_process_instance where instance_uid in (select instance_uid from old_instance);
delete from bpm_queue where status in ('failed', 'cancelled') and updated_at < now() - interval '14 days';
delete from bpm_message where expires_at < now();
commit;
SQL
