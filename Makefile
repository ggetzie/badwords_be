include .env

.PHONY: help
## help: Show this help message
help:
	@echo 'Usage:'
	@sed -n 's/^## //p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo 'Are you sure? [y/N]: \c' && read ans && [ $${ans:-N} = y ]

######################################################################
#                                                                    #
#                         Development                                #
#                                                                    #
######################################################################


## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	@go run ./cmd/api/ -db-dsn=${DATABASE_URL} -cors-trusted-origins="${CORS_TRUSTED_ORIGINS}"

## run/bw_chpwd: Change a user's password using the bw_chpwd command-line application
.PHONY: run/bw_chpwd
run/bw_chpwd:
	@go run ./cmd/cli/bw_chpwd/ -db-dsn="${DATABASE_URL}" -email="${EMAIL}" -new-password=${NEW_PW}

## run/bw_adduser: add a new user using the bw_adduser command-line application
.PHONY: run/bw_adduser
run/bw_adduser:
	@go run ./cmd/cli/bw_adduser/ -db-dsn="${DATABASE_URL}" -email="${EMAIL}" -password="${PASSWORD}" -full-name="${FULL_NAME}" -display-name="${DISPLAY_NAME}"

## run/token: generate a test Authentication token and save it to TOKEN
.PHONY: run/token
run/token:
	curl -d "{\"email\":\"${TEST_USER_EMAIL}\",\"password\":\"${TEST_USER_PASSWORD}\"}" http://localhost:8000/v1/tokens/authentication

######################################################################
#                                                                    #
#                         Database                                   #
#                                                                    #
######################################################################

## db/migrate/up: apply all up database migrations
.PHONY: db/migrate/up
db/migrate/up: confirm
	@echo "Applying all up database migrations"
	@migrate -path ./migrations -database ${DATABASE_URL} up

## db/migrate/down: apply 1 down database migration
.PHONY: db/migrate/down
db/migrate/down: confirm
	@echo "Applying 1 down migration"
	@migrate -path ./migrations -database ${DATABASE_URL} down 1

## db/migrate/create: create a new database migration with NAME
.PHONY: db/migrate/create
db/migrate/create:
	@echo "Creating a new database migration"
	@migrate create -ext sql -dir migrations -seq ${NAME}

## db/migrate/force: force database migrations to a specific VERSION
.PHONY: db/migrate/force
db/migrate/force: confirm
	@echo "Force migrations to ${VERSION}"
	@migrate -path ./migrations -database ${DATABASE_URL} force ${VERSION}	

## db/migrate/version: print the current database migration version and status
.PHONY: db/migrate/version
db/migrate/version:
	@echo "Current database migration version: \c"
	@migrate -path ./migrations -database ${DATABASE_URL} version

## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	@echo "Connecting to database ${DATABASE_URL}..."
	@psql ${DATABASE_URL}	

## db/backup: backup the database
.PHONY: db/backup
db/backup:
	@pg_dump ${DATABASE_URL} > backup/badwords_`date +%s`.pgsql

## db/random/puzzle: select a random puzzle from the database
.PHONY: db/random/puzzle
db/random/puzzle:
	@psql ${DATABASE_URL} -c "SELECT * FROM puzzles ORDER BY RANDOM() LIMIT 1"

## db/script: run a SQL script file with NAME from the scripts directory against the database
.PHONY: db/script 
db/script: confirm
	@psql ${DATABASE_URL} -f ./scripts/${NAME}

######################################################################
#                                                                    #
#                         Git                                        #
#                                                                    #
######################################################################	

## git/commit: commit changes to the repository
.PHONY: git/commit
git/commit:
	@go mod tidy
	@git status
	@git add -A .
	@git commit -m "${MSG}"
	@git status


######################################################################
#                                                                    #
#                         Build                                      #
#                                                                    #
######################################################################	

## build/api: build the cmd/api application
.PHONY: build/api
build/api:
	@echo "Building cmd/api..."
	@go build -ldflags='-s' -o=./bin/api ./cmd/api/
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/api ./cmd/api/
	
## build/bw_chpwd: build the cmd/chpwd application
.PHONY: build/bw_chpwd
build/bw_chpwd:
	@echo "Building cmd/chpwd..."
	@go build -ldflags='-s' -o=./bin/bw_chpwd ./cmd/cli/bw_chpwd/
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/bw_chpwd ./cmd/cli/bw_chpwd/

## build/bw_adduser: build the cmd/adduser application
.PHONY: build/bw_adduser
build/bw_adduser:
	@echo "Building cmd/adduser..."
	@go build -ldflags='-s' -o=./bin/bw_adduser ./cmd/cli/bw_adduser/
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/bw_adduser ./cmd/cli/bw_adduser/	


######################################################################
#                                                                    #
#                         Production                                 #
#                                                                    #
######################################################################	

production_host_ip = ${BADWORDS_IP}
BW_SSH = ssh -i ${BADWORDS_KEY}
BW_RSYNC = rsync -e "ssh -i ${BADWORDS_KEY}"
## production/connect: connect to the production server
.PHONY: production/connect
production/connect:
	@$(BW_SSH) badwords_user@${production_host_ip}

## production/deploy/api: deploy the cmd/api application to the production server
.PHONY: production/deploy/api
production/deploy/api: build/api
	$(BW_RSYNC) -P ./bin/linux_amd64/api badwords_user@${production_host_ip}:~
	$(BW_RSYNC) -rP --delete ./migrations badwords_user@${production_host_ip}:~
	$(BW_RSYNC) -P ./remote/api/production/badwords.service badwords_user@${production_host_ip}:~
	ssh -t -i ${BADWORDS_KEY} badwords_user@${production_host_ip} '\
	migrate -path ~/migrations -database $$BADWORDS_DB_DSN up \
	&& sudo mv ~/badwords.service /etc/systemd/system/badwords.service \
	&& sudo systemctl enable badwords \
	&& sudo systemctl restart badwords'

## production/scripts/api: copy scripts to the production server
.PHONY: production/scripts/api
production/scripts/api:
	$(BW_RSYNC) -rP --delete ./scripts badwords_user@${production_host_ip}:~
	

## production/update/api: update the cmd/api application on the production server
.PHONY: production/update/api
production/update/api: build/api
	$(BW_RSYNC) -P ./bin/linux_amd64/api badwords_user@${production_host_ip}:~
	$(BW_RSYNC) -rP --delete ./migrations badwords_user@${production_host_ip}:~
	$(BW_SSH) -t badwords_user@${production_host_ip} '\
	migrate -path ~/migrations -database $$BADWORDS_DB_DSN up \
	&& sudo systemctl restart badwords'

## production/deploy/bw_chpwd: deploy the cmd/chpwd application to the production server
.PHONY: production/deploy/bw_chpwd
production/deploy/bw_chpwd: build/bw_chpwd
	$(BW_RSYNC) -P ./bin/linux_amd64/bw_chpwd badwords_user@${production_host_ip}:~
	
	
## production/setup/api: copy setup scripts to the production server
.PHONY: production/setup/api
production/setup/api:
	$(BW_RSYNC) -rP --delete ./remote/api/setup badwords_user@${production_host_ip}:~

## production/nginx/api: copy nginx configuration to the production server
.PHONY: production/nginx/api
production/nginx/api:
	$(BW_RSYNC) -rP --delete ./remote/api/production/nginx_badwords_80.conf badwords_user@${production_host_ip}:~/nginx_badwords.conf
	$(BW_SSH) -t badwords_user@${production_host_ip} '\
	sudo mv ~/nginx_badwords.conf /etc/nginx/sites-available/nginx_badwords.conf \
	&& sudo ln -sf /etc/nginx/sites-available/nginx_badwords.conf /etc/nginx/sites-enabled/nginx_badwords.conf \
	&& sudo systemctl restart nginx'	
