build:
	#./hack/update-codegen.sh
	rm -f xworld-controller
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o xworld-controller main.go controller.go
	rm -rf /home/xx/k8s/xworld/*
	cp ./* -r /home/xx/k8s/xworld/
	chown xx -R /home/xx/k8s/xworld/*
	chgrp xx -R /home/xx/k8s/xworld/*

install:
	kubectl delete crd xs
	docker build ./ --tag=xworld-controller:0.0.1
