### Source Code for the great talk [Practical Memory Profiling](https://www.youtube.com/watch?v=6qAfkJGWsns) from Bill Kennedy

# Benchmark

```sh
# "-gcflags -m=2" show escape analysis result
$ go test -bench . -benchmem --memprofile p.out -gcflags -m=2
```

```sh
# Run profiler in command prompt
$ go tool pprof p.out
File: go-rpc.test
Type: alloc_space
Time: Nov 9, 2024 at 3:08pm (CST)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) list algoOne
Total: 116.50MB
ROUTINE ======================== go-rpc.algoOne in /Users/yitao/go-rpc/main.go
   13.50MB   116.50MB (flat, cum)   100% of Total
         .          .     64:func algoOne(data []byte, find []byte, repl []byte, output *bytes.Buffer) {
         .          .     65:   // use a bytes buffer to provide a stream to process
         .      103MB     66:   input := bytes.NewBuffer(data)
         .          .     67:
         .          .     68:   // the number of bytes we are looking for
         .          .     69:   size := len(find)
         .          .     70:
         .          .     71:   // declare the buffers we need to process the stream.
   13.50MB    13.50MB     72:   buf := make([]byte, size)
         .          .     73:   end := size - 1
         .          .     74:
         .          .     75:   // Read in an initial number of bytes we need to get started
         .          .     76:   if n, err := io.ReadFull(input, buf[:end]); err != nil {
         .          .     77:           output.Write(buf[:n])
(pprof)

```

There are two columns: `flat`, `cum` (accumulative)

- `flat`: allocation happening because of code inside the functions being called here
- `cum`: allocation happening down the call path

line `main.go:66` does not have flat allocation, means allocation happens only down the call path

Now let's see the result from the web

```sh
# Run profiler in browser
$ go tool pprof -http=:6060 p.out
```

![memprofile](./memprofile.png)

web version is different from the console version that web version showing that allocation is flat (only happens at bytes.NewBuffer func call), whereas console version showing that allocation is accumlative (down the call path).

Let's see why by showing escape analysis result.

```sh
$ go tool pprof p.out
File: go-rpc.test
Type: alloc_space
Time: Nov 9, 2024 at 3:08pm (CST)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) o
  call_tree                 = false
  compact_labels            = true
  divide_by                 = 1
  drop_negative             = false
  edgefraction              = 0.001
  focus                     = ""
  granularity               = functions            //: [addresses | filefunctions | files | functions | lines]
  hide                      = ""
  ignore                    = ""
  intel_syntax              = false
  mean                      = false
  nodecount                 = -1                   //: default
  nodefraction              = 0.005
  noinlines                 = false
  normalize                 = false
  output                    = ""
  prune_from                = ""
  relative_percentages      = false
  sample_index              = inuse_space          //: [alloc_objects | alloc_space | inuse_objects | inuse_space]
  show                      = ""
  show_from                 = ""
  sort                      = flat                 //: [cum | flat]
  tagfocus                  = ""
  taghide                   = ""
  tagignore                 = ""
  tagleaf                   = ""
  tagroot                   = ""
  tagshow                   = ""
  trim                      = true
  trim_path                 = ""
  unit                      = minimum
(pprof)
```

we can see `noinlines                 = false` in console version.

However, in web version, `noinlines = true`.

> Inlining optimization: no calling the function, instead copy the function content over here inline.

Let's change it to true, and `list algoOne` again:

```sh
(pprof) noinlines = true
(pprof) list algoOne
Total: 116.50MB
ROUTINE ======================== go-rpc.algoOne in /Users/yitao/go-rpc/main.go
  116.50MB   116.50MB (flat, cum)   100% of Total
         .          .     64:func algoOne(data []byte, find []byte, repl []byte, output *bytes.Buffer) {
         .          .     65:   // use a bytes buffer to provide a stream to process
     103MB      103MB     66:   input := bytes.NewBuffer(data)
         .          .     67:
         .          .     68:   // the number of bytes we are looking for
         .          .     69:   size := len(find)
         .          .     70:
         .          .     71:   // declare the buffers we need to process the stream.
   13.50MB    13.50MB     72:   buf := make([]byte, size)
         .          .     73:   end := size - 1
         .          .     74:
         .          .     75:   // Read in an initial number of bytes we need to get started
         .          .     76:   if n, err := io.ReadFull(input, buf[:end]); err != nil {
         .          .     77:           output.Write(buf[:n])
(pprof)
```

Now we can see there is a flat allocation in console version.

`buffer.NewBuffer` will allocation on heap because it returns a pointer to local variable, that local variable needs to escape to heap to outlive the its enclosing function.

```go
func NewBuffer(buf []byte) *Buffer { return &Buffer{buf: buf }
```

But thanks to inline optimisation:

```go
input := bytes.NewBuffer(data)

// will become

input := &Buffer{buf: data}
```

Therefore, no need to escape and allocate.

> Inlining rule: generally if a function is simple enough and does not call any other functions (it is a leaf function), it can be inlined.

```
./main.go:107:6: cannot inline algoTwo: function too complex: cost 309 exceeds budget 80
```

```sh
$ go test -bench . -benchmem --memprofile p.out -gcflags -m=2
...
./algoOne.go:10:26: &bytes.Buffer{...} escapes to heap:
./algoOne.go:10:26:   flow: ~r0 = &{storage for &bytes.Buffer{...}}:
./algoOne.go:10:26:     from &bytes.Buffer{...} (spill) at ./algoOne.go:10:26
./algoOne.go:10:26:     from ~r0 = &bytes.Buffer{...} (assign-pair) at ./algoOne.go:10:26
./algoOne.go:10:26:   flow: input = ~r0:
./algoOne.go:10:26:     from input := ~r0 (assign) at ./algoOne.go:10:8
./algoOne.go:10:26:   flow: io.r = input:
./algoOne.go:10:26:     from input (interface-converted) at ./algoOne.go:36:29
./algoOne.go:10:26:     from io.r, io.buf := input, buf[:end] (assign-pair) at ./algoOne.go:36:28
./algoOne.go:10:26:   flow: {heap} = io.r:
./algoOne.go:10:26:     from io.ReadAtLeast(io.r, io.buf, len(io.buf)) (call parameter) at ./algoOne.go:36:28
./algoOne.go:8:14: parameter data leaks to {storage for &bytes.Buffer{...}} with derefs=0:
./algoOne.go:8:14:   flow: bytes.buf = data:
./algoOne.go:8:14:     from bytes.buf := data (assign-pair) at ./algoOne.go:10:26
./algoOne.go:8:14:   flow: {storage for &bytes.Buffer{...}} = bytes.buf:
./algoOne.go:8:14:     from bytes.Buffer{...} (struct literal element) at ./algoOne.go:10:26
...
```

Notice that `&bytes.Buffer{...} escapes to heap:` and `from input (interface-converted) at ./algoOne.go:36:29`

func `io.ReadFull` takes interface type `Reader` as first argument:
`func ReadFull(r Reader, buf []byte) (n int, err error) {` and this will cause one allocation.

_Converting a value to an interface in Go often leads to an allocation on the heap_. Here's why:
Interface Internals
An interface value in Go is represented internally as a pair of values:
A pointer to the underlying concrete type's data.
A pointer to the type's method table.
Allocation When Converting
When you convert a value to an interface, Go needs to create this interface value pair on the heap.
This is because the interface value needs to store the type information of the concrete value, which can't be determined at compile time.

Let's change `io.ReadFull` to `input.Read(buf[:end])` and run benchmark again:

```sh
$ go test -bench . -benchmem --memprofile p.out
goos: darwin
goarch: arm64
pkg: practical-memory-profiling
BenchmarkAlgorithmOne-8                  1160020              1025 ns/op              53 B/op          2 allocs/op
BenchmarkAlgorithmTwo-8                  4278295               280.5 ns/op             0 B/op          0 allocs/op
BenchmarkAlgorithmOneVersion2-8          1534524               905.3 ns/op             5 B/op          1 allocs/op
PASS
ok      practical-memory-profiling      7.108s
```

Memory allocation is reduced down to 1.

Pprof with noinlines: `go tool pprof -noinlines p.out`

```sh
$ go tool pprof -noinlines p.out
File: practical-memory-profiling.test
Type: alloc_space
Time: Nov 9, 2024 at 3:40pm (CST)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) list algoOneVersion2
Total: 115MB
ROUTINE ======================== practical-memory-profiling.algoOneVersion2 in /Users/yitao/practical-memory-profiling/algoOneVersion2.go
      15MB       15MB (flat, cum) 13.04% of Total
         .          .      7:func algoOneVersion2(data []byte, find []byte, repl []byte, output *bytes.Buffer) {
         .          .      8:   // use a bytes buffer to provide a stream to process
         .          .      9:   input := bytes.NewBuffer(data)
         .          .     10:
         .          .     11:   // the number of bytes we are looking for
         .          .     12:   size := len(find)
         .          .     13:
         .          .     14:   // declare the buffers we need to process the stream.
      15MB       15MB     15:   buf := make([]byte, size)
         .          .     16:   end := size - 1
         .          .     17:
         .          .     18:   // Read in an initial number of bytes we need to get started
         .          .     19:   if n, err := input.Read(buf[:end]); err != nil {
         .          .     20:           output.Write(buf[:n])
(pprof)
```

we still have one allocation `buf := make([]byte, size)`

search through the escape analysis result we can find:

```sh
$ go test -bench . -benchmem --memprofile p.out -gcflags -m=2
...
./algoOneVersion2.go:15:13: make([]byte, size) escapes to heap:
./algoOneVersion2.go:15:13:   flow: {heap} = &{storage for make([]byte, size)}:
./algoOneVersion2.go:15:13:     from make([]byte, size) (non-constant size) at ./algoOneVersion2.go:15:13
...
```

change `buf := make([]byte, size)` to `buf := make([]byte, 5)` will reduce one allocation.

```sh
$ go test -bench . -benchmem --memprofile p.out
pkg: practical-memory-profiling
BenchmarkAlgorithmOne-8                  1169751              1037 ns/op              53 B/op          2 allocs/op
BenchmarkAlgorithmTwo-8                  4195002               292.6 ns/op             0 B/op          0 allocs/op
BenchmarkAlgorithmOneVersion2-8          1874484               629.8 ns/op             0 B/op          0 allocs/op
PASS
ok      practical-memory-profiling      7.168s
```

But algoOneVersion2 is still quite slow. Let's now see cpu profile:

```sh
$ go test -bench . -benchmem --cpuprofile cpu.out
goos: darwin
goarch: arm64
pkg: practical-memory-profiling
BenchmarkAlgorithmOne-8                  1151667              1020 ns/op              53 B/op          2 allocs/op
BenchmarkAlgorithmTwo-8                  4158492               283.9 ns/op             0 B/op          0 allocs/op
BenchmarkAlgorithmOneVersion2-8          1850841               638.4 ns/op             0 B/op          0 allocs/op
PASS
ok      practical-memory-profiling      6.401s
```

```sh
$ go tool pprof cpu.out
File: practical-memory-profiling.test
Type: cpu
Time: Nov 9, 2024 at 3:48pm (CST)
Duration: 5.25s, Total samples = 4.54s (86.49%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) list algoOneVersion2
Total: 4.54s
ROUTINE ======================== practical-memory-profiling.algoOneVersion2 in /Users/yitao/practical-memory-profiling/algoOneVersion2.go
     320ms      1.63s (flat, cum) 35.90% of Total
         .          .      7:func algoOneVersion2(data []byte, find []byte, repl []byte, output *bytes.Buffer) {
         .          .      8:   // use a bytes buffer to provide a stream to process
         .          .      9:   input := bytes.NewBuffer(data)
         .          .     10:
         .          .     11:   // the number of bytes we are looking for
         .          .     12:   size := len(find)
         .          .     13:
         .          .     14:   // declare the buffers we need to process the stream.
         .          .     15:   buf := make([]byte, 5)
         .          .     16:   end := size - 1
         .          .     17:
         .          .     18:   // Read in an initial number of bytes we need to get started
         .          .     19:   if n, err := input.Read(buf[:end]); err != nil {
      60ms       60ms     20:           output.Write(buf[:n])
         .          .     21:           return
         .          .     22:   }
         .          .     23:
         .          .     24:   for {
         .          .     25:           // read in one byte from the input stream
      10ms      510ms     26:           if _, err := input.Read(buf[end:]); err != nil {
      10ms       10ms     27:                   output.Write(buf[:end])
         .          .     28:                   return
         .          .     29:           }
         .          .     30:
         .          .     31:           // if we have a match, replace the bytes
         .      400ms     32:           if bytes.Equal(buf, find) {
     230ms      310ms     33:                   output.Write(repl)
         .          .     34:                   // read a new initial number of bytes
         .       80ms     35:                   if n, err := input.Read(buf[:end]); err != nil {
         .       20ms     36:                           output.Write(buf[:n])
      10ms       10ms     37:                           return
         .          .     38:                   }
         .          .     39:                   continue
         .          .     40:           }
         .          .     41:
         .          .     42:           // write the front byte since it has been compared
         .      230ms     43:           output.WriteByte(buf[0])
         .          .     44:           // slice that front byte out
         .          .     45:           copy(buf, buf[1:])
         .          .     46:   }
         .          .     47:}
(pprof)
```

this line takes most of the time:

```
         .      400ms     32:           if bytes.Equal(buf, find) {
```

Now you have found the bottleneck, you can of course keep optimizing this specifc function if you need using the techniques we have used above.
