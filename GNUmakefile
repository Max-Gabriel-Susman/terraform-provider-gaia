default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

update_docs:
	curl  --output docs/index.md https://raw.githubusercontent.com/Max-Gabriel-Susman/terraform-provider-ouroboros/main/docs/index.md && \
	curl  --output docs/data-sources/order.md  https://raw.githubusercontent.com/Max-Gabriel-Susman/terraform-provider-ouroboros/main/docs/data-sources/scaffolding_data_source.md && \
	curl  --output docs/resources/order.md https://raw.githubusercontent.com/Max-Gabriel-Susman/terraform-provider-ouroboros/main/docs/resources/scaffolding_resource.md
