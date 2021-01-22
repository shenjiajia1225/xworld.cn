build:
	#./hack/update-codegen.sh
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o world-controller main.go controller.go
	rm -rf /home/xx/k8s/xworld/*
	cp ./* -r /home/xx/k8s/xworld/
	chown xx -R /home/xx/k8s/xworld/*
	chgrp xx -R /home/xx/k8s/xworld/*

install:
	kubectl delete crd xs
	docker build ./ --tag=xworld-controller:0.0.1
