build:
	mkdir -p build
	rustc --out-dir build -O src/syslog/lib.rs

test: build
	rustc -L build -o build/test --test src/syslog/test.rs
	./build/test

examples: build
	rustc -L build -o build/example examples/write.rs
	./build/example

clean:
	rm -rf build
