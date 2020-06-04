include .bingo/Variables.mk

faillint-exists: $(FAILLINT)
	@echo "checking faillint"
	@test $(FAILLINT) || (echo "no var" && exit 1)
	@test -f $(FAILLINT) || (echo "no file" && exit 1)

goimports-exists: $(GOIMPORTS)
	@echo "checking goimports"
	@test $(GOIMPORTS) || (echo "no var" && exit 1)
	@test -f $(GOIMPORTS) || (echo "no file" && exit 1)

goimports2-exists: $(GOIMPORTS2)
	@echo "checking goimports2"
	@test $(GOIMPORTS2) || (echo "no var" && exit 1)
	@test -f $(GOIMPORTS2) || (echo "no file" && exit 1)
