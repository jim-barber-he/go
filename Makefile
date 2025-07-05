.PHONY: all build clean install lint lintall run vet

all: vet lint build install

build clean install run:
	$(MAKE) -C golock $@
	$(MAKE) -C kubectl-plugins/kubectl-n $@
	$(MAKE) -C kubectl-plugins/kubectl-p $@
	$(MAKE) -C ssm $@

lint lintall vet:
	$(MAKE) -C aws $@
	$(MAKE) -C golock $@
	$(MAKE) -C k8s $@
	$(MAKE) -C kubectl-plugins/kubectl-n $@
	$(MAKE) -C kubectl-plugins/kubectl-p $@
	$(MAKE) -C ssm $@
	$(MAKE) -C texttable $@
	$(MAKE) -C util $@

modules:
	go get go@latest
	go get -u ./...
	go mod tidy
