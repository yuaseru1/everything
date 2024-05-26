dev:
	@which reflex>/dev/null || go install github.com/cespare/reflex@latest
	reflex -s -d none -r '.(go|js|css|html)$$' -- make run

run:
	go run main.go
