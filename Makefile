.PHONY: all build clean install lint run vet

all: vet lint build install

build clean install lint lintall run vet:
	$(MAKE) -C kubectl-plugins/kubectl-n $@
