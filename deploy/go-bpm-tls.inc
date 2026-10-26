# nginx: сертификат и TLS сайта go-bpm.tech. На сервере - /etc/nginx/go-bpm-tls.inc.
ssl_certificate /etc/letsencrypt/live/go-bpm.tech/fullchain.pem;
ssl_certificate_key /etc/letsencrypt/live/go-bpm.tech/privkey.pem;
ssl_session_cache shared:go_bpm_ssl:5m;
ssl_session_timeout 1d;
