mkdir -p /home/badwords_user/logs
touch /home/badwords_user/logs/nginx-access.log
touch /home/badwords_user/logs/nginx-error.log
touch /home/badwords_user/logs/badwords.log

sudo chown www-data:www-data /home/badwords_user/logs/nginx-access.log
sudo chown www-data:www-data /home/badwords_user/logs/nginx-error.log