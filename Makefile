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
	@go run ./cmd/api/ -db-dsn=${DATABASE_URL} -mail-sender=${SERVER_EMAIL} \
	-aws-access-key-id=${AWS_ACCESS_KEY_ID} -aws-secret-access-key=${AWS_SECRET_ACCESS_KEY} \
	-openai-key="${OPENAI_API_KEY}" -openai-endpoint="${OPENAI_ENDPOINT}" -openai-model="${OPENAI_MODEL}" \
	-cors-trusted-origins="${CORS_TRUSTED_ORIGINS}" -proxy-url="${PROXY_URL}"

## run/vectorize: run the cmd/cli/vectorize application
.PHONY: run/vectorize 
run/vectorize:
	@go run ./cmd/cli/vectorize/ -db-dsn="${DATABASE_URL}" -openai-key="${OPENAI_API_KEY}" -proxy-url="${PROXY_URL}"

## run/snpw: run the cmd/snpw application
.PHONY: run/snpw
run/snpw:
	@go run ./cmd/cli/snpw/ -db-dsn="${DATABASE_URL}" -email="${EMAIL}" -new-password=${NEW_PW}

## run/eg_proj_created: run the cmd/cli/engenium/projectCreated
.PHONY: run/eg_proj_created
run/eg_proj_created:
	@go run ./cmd/cli/engenium/projectCreated/

## run/cors/preflight: run the	cmd/cors/preflight application
.PHONY: run/cors/preflight
run/cors/preflight:
	@go run ./cmd/cors/preflight/ -email="${TEST_USER_EMAIL}" -password="${TEST_USER_PASSWORD}"

## run/cors/simple: run the	cmd/cors/simple application
.PHONY: run/cors/simple
run/cors/simple:
	@go run ./cmd/cors/simple/ -addr=":9001"

## run/token: generate a test Authentication token and save it to TOKEN
.PHONY: run/token
run/token:
	TOKEN=$(shell curl -d '{"email":"${TEST_USER_EMAIL}","password":"${TEST_USER_PASSWORD}"}' http://localhost:8000/v1/tokens/authentication | jq -r .authentication_token.token)

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
	@psql ${DATABASE_URL}	

## db/backup: backup the database
.PHONY: db/backup
db/backup:
	@pg_dump ${DATABASE_URL} > backup/supernotes_`date +%s`.pgsql

## db/random/keynote: select a random keynote from the database
.PHONY: db/random/keynote
db/random/keynote:
	@psql ${DATABASE_URL} -c "SELECT * FROM keynotes ORDER BY RANDOM() LIMIT 1"

## db/random/project: select a random project from the database
.PHONY: db/random/project
db/random/project:
	@psql ${DATABASE_URL} -c "SELECT p.id, p.number, p.name FROM projects p INNER JOIN project_keynotes pk ON p.id = pk.project_id ORDER BY random() LIMIT 1"	

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

## build/snpw: build the cmd/snpw application
.PHONY: build/snpw
build/snpw:
	@echo "Building cmd/snpw..."
	@go build -ldflags='-s' -o=./bin/snpw ./cmd/cli/snpw/
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/snpw ./cmd/cli/snpw/

## build/eg_sync: build the cmd/cli/engenium/sync application
.PHONY: build/eg_sync
build/eg_sync:
	@echo "Building cmd/cli/engenium/sync..."
	GOOS=windows GOARCH=amd64 go build -ldflags='-s' -o=./bin/windows_amd64/eg_sync ./cmd/cli/engenium/sync/


######################################################################
#                                                                    #
#                         Production                                 #
#                                                                    #
######################################################################	

production_host_ip = ${SUPERNOTES_IP}
SN_SSH = ssh -i ${SUPERNOTES_KEY}
SN_RSYNC = rsync -e "ssh -i ${SUPERNOTES_KEY}"
## production/connect: connect to the production server
.PHONY: production/connect
production/connect:
	@$(SN_SSH) supernotes_user@${production_host_ip}

## production/deploy/api: deploy the cmd/api application to the production server
.PHONY: production/deploy/api
production/deploy/api: build/api
	$(SN_RSYNC) -P ./bin/linux_amd64/api supernotes_user@${production_host_ip}:~
	$(SN_RSYNC) -rP --delete ./migrations supernotes_user@${production_host_ip}:~
	$(SN_RSYNC) -P ./remote/api/production/supernotes.service supernotes_user@${production_host_ip}:~
	ssh -t -i ${SUPERNOTES_KEY} supernotes_user@${production_host_ip} '\
	migrate -path ~/migrations -database $$SUPERNOTES_DB_DSN up \
	&& sudo mv ~/supernotes.service /etc/systemd/system/supernotes.service \
	&& sudo systemctl enable supernotes \
	&& sudo systemctl restart supernotes'

## production/scripts/api: copy scripts to the production server
.PHONY: production/scripts/api
production/scripts/api:
	$(SN_RSYNC) -rP --delete ./scripts supernotes_user@${production_host_ip}:~
	

## production/update/api: update the cmd/api application on the production server
.PHONY: production/update/api
production/update/api: build/api
	$(SN_RSYNC) -P ./bin/linux_amd64/api supernotes_user@${production_host_ip}:~
	$(SN_RSYNC) -rP --delete ./migrations supernotes_user@${production_host_ip}:~
	$(SN_SSH) -t supernotes_user@${production_host_ip} '\
	migrate -path ~/migrations -database $$SUPERNOTES_DB_DSN up \
	&& sudo systemctl restart supernotes'

## production/deploy/snpw: deploy the cmd/snpw application to the production server
.PHONY: production/deploy/snpw
production/deploy/snpw: build/snpw
	$(SN_RSYNC) -P ./bin/linux_amd64/snpw supernotes_user@${production_host_ip}:~
	
	
## production/setup/api: copy setup scripts to the production server
.PHONY: production/setup/api
production/setup/api:
	$(SN_RSYNC) -rP --delete ./remote/api/setup supernotes_user@${production_host_ip}:~

## production/nginx/api: copy nginx configuration to the production server
.PHONY: production/nginx/api
production/nginx/api:
	$(SN_RSYNC) -rP --delete ./remote/api/production/nginx_supernotes_80.conf supernotes_user@${production_host_ip}:~/nginx_supernotes.conf
	$(SN_SSH) -t supernotes_user@${production_host_ip} '\
	sudo mv ~/nginx_supernotes.conf /etc/nginx/sites-available/nginx_supernotes.conf \
	&& sudo ln -sf /etc/nginx/sites-available/nginx_supernotes.conf /etc/nginx/sites-enabled/nginx_supernotes.conf \
	&& sudo systemctl restart nginx'	

## production/deploy/web: deploy the web frontend to S3
.PHONY: production/deploy/web
production/deploy/web:
	@echo "Deploying web frontend to S3..."
	cd frontend && npm run build
	@aws --profile supernotes s3 sync ./frontend/dist/ s3://${AWS_AMPLIFY_S3_BUCKET} --delete 
	@aws --profile supernotes amplify start-deployment --app-id ${AWS_AMPLIFY_APP_ID} --branch-name main \
	--source-url s3://${AWS_AMPLIFY_S3_BUCKET} --source-url-type BUCKET_PREFIX

## production/check/web: check the status of the web frontend deployment
.PHONY: production/check/web
production/check/web:
	@aws --profile supernotes amplify get-job --app-id ${AWS_AMPLIFY_APP_ID} --branch-name main --job-id ${JOB}

## production/deploy/eg_sync: build the cmd/cli/engenium/sync application and copy to OneDrive
.PHONY: production/deploy/eg_sync
production/deploy/eg_sync: build/eg_sync
	@cp ./bin/windows_amd64/eg_sync ~/OneDrive\ -\ Engenium\ Group/Documents/keynotes/eg_sync.exe
	@echo "Done!"
	