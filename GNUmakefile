default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v -tags=acceptance $(TESTARGS) -timeout 120m
