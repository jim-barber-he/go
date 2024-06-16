.PHONY: all build clean install lint lintall run vet

all: vet lint build install

build clean install run:
	$(MAKE) -C kubectl-plugins/kubectl-n $@
	$(MAKE) -C kubectl-plugins/kubectl-p $@

lint lintall vet:
	$(MAKE) -C k8s $@
	$(MAKE) -C kubectl-plugins/kubectl-n $@
	$(MAKE) -C kubectl-plugins/kubectl-p $@
	$(MAKE) -C texttable $@
	$(MAKE) -C util $@
