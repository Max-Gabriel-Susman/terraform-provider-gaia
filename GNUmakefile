default: testacc

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

# update documentation to the main branch of the repo
update_docs:
	curl  --output docs/index.md https://raw.githubusercontent.com/Max-Gabriel-Susman/terraform-provider-ouroboros/main/docs/index.md && \
	curl  --output docs/data-sources/scaffolding_data_source.md  https://raw.githubusercontent.com/Max-Gabriel-Susman/terraform-provider-ouroboros/main/docs/data-sources/scaffolding_data_source.md && \
	curl  --output docs/resources/scaffolding_resource.md https://raw.githubusercontent.com/Max-Gabriel-Susman/terraform-provider-ouroboros/main/docs/resources/scaffolding_resource.md

# add goreleaser config from terraform-provider-scaffolding
add_goreleaser_config:
	curl --output .goreleaser.yml https://raw.githubusercontent.com/hashicorp/terraform-provider-scaffolding/main/.goreleaser.yml
