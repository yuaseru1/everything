dev:
	@which reflex>/dev/null || go install github.com/cespare/reflex@latest
	reflex -s -d none -r '.(go|js|css|html)$$' -- make run

run:
	env AWS_BUCKET=nervos AWS_ACCESS_KEY=$(HABARI_AWS_ACCESS_KEY) AWS_SECRET_KEY=$(HABARI_AWS_SECRET_KEY) go run main.go

deploy:
	echo "railway"
