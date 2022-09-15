build:
	docker build -t songjiayang/mimir-otlp:0.0.1 .
start:
	docker-compose up -d 
stop:
	docker-compose down