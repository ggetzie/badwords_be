# set a cron job to delete expired tokens every day at 3AM
(crontab -l 2>/dev/null; echo "0 3 * * * psql $BW_DATABASE_URL -c \"DELETE FROM tokens WHERE expiry < now();\"") | crontab -