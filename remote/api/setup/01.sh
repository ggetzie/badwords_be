#!/bin/bash

set -eu
USERNAME="badwords_user"
read -p "Enter the password for the badwords DB user: " DB_PASSWORD
echo "DB_PASSWORD=${DB_PASSWORD}"

# confirm the password is correct before continuing
read -p "Is the password correct? (y/n): " CONFIRM
if [[ ${CONFIRM} != "y" ]]; then
  echo "Exiting..."
  exit 1
fi

# Add new user with sudo privileges if it doesn't exist
if ! id -u ${USERNAME}; then
  useradd --create-home --shell "/bin/bash" --groups sudo "${USERNAME}"
  # Force user to change password on first login
  passwd --delete ${USERNAME}
  chage --lastday 0 ${USERNAME}
  rsync --archive --chown=${USERNAME}:${USERNAME} /home/gabe/.ssh /home/${USERNAME}
fi

# install the migrate tool
EXTRACT_DIR="/tmp/migrate"
mkdir -p ${EXTRACT_DIR}
curl -o /tmp/migrate.linux-amd64.tar.gz  -L https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-amd64.tar.gz 
tar -xvf /tmp/migrate.linux-amd64.tar.gz -C ${EXTRACT_DIR}
mv ${EXTRACT_DIR}/migrate /usr/local/bin/migrate
rm -r /tmp/migrate.linux-amd64.tar.gz
rm -r ${EXTRACT_DIR}

# Set up the database
sudo -i -u postgres psql -c "CREATE DATABASE badwords_be;"
sudo -i -u postgres psql badwords_be -c "CREATE USER ${USERNAME} WITH PASSWORD '${DB_PASSWORD}';"
