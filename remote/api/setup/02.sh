#! /bin/bash

# run locally

# get the latest backup file
BACKUP_FOLDER="/usr/local/src/badwords_be/backup"
LATEST_BACKUP=$(ls -t ${BACKUP_FOLDER} | head -n 1)

echo "LATEST_BACKUP=${LATEST_BACKUP}"

# transfer the backup file to the remote server
scp -i ${HETZNER_KEY} ${BACKUP_FOLDER}/${LATEST_BACKUP} badwords_user@${HETZNER_GEN_IP}:badwords_be.pgsql

# restore the backup file on the remote server
ssh -t -i ${HETZNER_KEY} badwords_user@${HETZNER_GEN_IP} "sudo -i -u postgres psql badwords_be < badwords_be.pgsql"

