
build:
	cd go && go mod download && go build -o ../pollinate .

run: build
	./pollinate

build-docker:
	docker build -t maplesond/pollinate:latest .

run-docker: build-docker
	docker run -p8000:8000 maplesond/pollinate:latest

publish: build-docker
	docker push maplesond/pollinate:latest

NAMESPACE:="pollinate"
deploy:
	cd helm && helm dependency build pollinate && helm upgrade --install --create-namespace -n ${NAMESPACE} pollinate pollinate