#!/bin/bash
set -e

echo "🚀 Starting optimized version testing..."
echo ""

if [ ! -f "benchmarks/baseline.txt" ]; then
    echo "⚠️  Baseline results not found!"
    echo "First run: ./scripts/run_baseline.sh"
    exit 1
fi

mkdir -p benchmarks profiles results

echo "📡 Checking service availability..."
if ! curl -s http://localhost:8080/health > /dev/null; then
    echo "❌ Service unavailable at http://localhost:8080"
    echo "Start the service: docker-compose up -d"
    exit 1
fi
echo "✅ Service available"
echo ""

echo "🔬 Running Go benchmarks..."
go test -bench=. -benchmem ./... | tee benchmarks/optimized.txt
echo ""

echo "⚡ Load test: Create (1000 requests, 10 concurrent)..."
go run scripts/load_test.go \
    -url http://localhost:8080 \
    -requests 1000 \
    -concurrency 10 \
    -operation create \
    | tee results/optimized_create.txt
echo ""

echo "⚡ Load test: Get (2000 requests, 20 concurrent)..."
go run scripts/load_test.go \
    -url http://localhost:8080 \
    -requests 2000 \
    -concurrency 20 \
    -operation get \
    | tee results/optimized_get.txt
echo ""

echo "⚡ Load test: Mixed (1000 requests, 15 concurrent)..."
go run scripts/load_test.go \
    -url http://localhost:8080 \
    -requests 1000 \
    -concurrency 15 \
    -operation mixed \
    | tee results/optimized_mixed.txt
echo ""

echo "🔍 Collecting CPU profile (30 seconds)..."
echo "Starting background load..."
go run scripts/load_test.go \
    -url http://localhost:8080 \
    -duration 35s \
    -concurrency 20 \
    -operation mixed > /dev/null 2>&1 &
LOAD_PID=$!

sleep 2
curl -s http://localhost:6060/debug/pprof/profile?seconds=30 > profiles/optimized_cpu.prof
wait $LOAD_PID
echo "✅ CPU profile saved: profiles/optimized_cpu.prof"
echo ""

echo "🔍 Collecting Heap profile..."
curl -s http://localhost:6060/debug/pprof/heap > profiles/optimized_heap.prof
echo "✅ Heap profile saved: profiles/optimized_heap.prof"
echo ""

echo "🔍 Collecting Goroutine profile..."
curl -s http://localhost:6060/debug/pprof/goroutine > profiles/optimized_goroutine.prof
echo "✅ Goroutine profile saved: profiles/optimized_goroutine.prof"
echo ""

echo "📊 Collecting Database Pool metrics..."
curl -s http://localhost:8080/metrics/db | jq '.' | tee results/optimized_db_metrics.json
echo ""

echo "════════════════════════════════════════════════════"
echo "📊 RESULTS COMPARISON"
echo "════════════════════════════════════════════════════"
echo ""

if command -v benchstat &> /dev/null; then
    echo "📈 Go benchmarks comparison:"
    echo "---------------------------------------------------"
    benchstat benchmarks/baseline.txt benchmarks/optimized.txt
    echo ""
else
    echo "⚠️  benchstat not installed. Install: go install golang.org/x/perf/cmd/benchstat@latest"
    echo ""
fi

echo "📈 Load tests comparison:"
echo "---------------------------------------------------"
echo ""
echo "CREATE operation:"
echo "  Baseline:"
grep "Throughput" results/baseline_create.txt || echo "  No data"
grep "Average" results/baseline_create.txt || echo "  No data"
echo "  Optimized:"
grep "Throughput" results/optimized_create.txt || echo "  No data"
grep "Average" results/optimized_create.txt || echo "  No data"
echo ""

echo "GET operation:"
echo "  Baseline:"
grep "Throughput" results/baseline_get.txt || echo "  No data"
grep "Average" results/baseline_get.txt || echo "  No data"
echo "  Optimized:"
grep "Throughput" results/optimized_get.txt || echo "  No data"
grep "Average" results/optimized_get.txt || echo "  No data"
echo ""

echo "MIXED operation:"
echo "  Baseline:"
grep "Throughput" results/baseline_mixed.txt || echo "  No data"
grep "Average" results/baseline_mixed.txt || echo "  No data"
echo "  Optimized:"
grep "Throughput" results/optimized_mixed.txt || echo "  No data"
grep "Average" results/optimized_mixed.txt || echo "  No data"
echo ""

echo "📈 Database Pool metrics comparison:"
echo "---------------------------------------------------"
echo "  Baseline wait_count: $(jq -r '.wait_count' results/baseline_db_metrics.json 2>/dev/null || echo 'N/A')"
echo "  Optimized wait_count: $(jq -r '.wait_count' results/optimized_db_metrics.json 2>/dev/null || echo 'N/A')"
echo ""
echo "  Baseline wait_duration: $(jq -r '.wait_duration' results/baseline_db_metrics.json 2>/dev/null || echo 'N/A')"
echo "  Optimized wait_duration: $(jq -r '.wait_duration' results/optimized_db_metrics.json 2>/dev/null || echo 'N/A')"
echo ""

echo "📈 CPU profiles analysis:"
echo "---------------------------------------------------"
if command -v go &> /dev/null; then
    echo "Top 5 functions (Baseline):"
    go tool pprof -top -nodecount=5 profiles/baseline_cpu.prof 2>/dev/null | grep -E "flat|cum|main\." || echo "  No data"
    echo ""
    echo "Top 5 functions (Optimized):"
    go tool pprof -top -nodecount=5 profiles/optimized_cpu.prof 2>/dev/null | grep -E "flat|cum|main\." || echo "  No data"
else
    echo "  Go not found, skipping profile analysis"
fi
echo ""

echo "════════════════════════════════════════════════════"
echo "✅ Optimized version testing completed!"
echo ""
echo "Results saved in:"
echo "  📁 benchmarks/optimized.txt          - Go benchmarks"
echo "  📁 results/optimized_create.txt      - Create load test"
echo "  📁 results/optimized_get.txt         - Get load test"
echo "  📁 results/optimized_mixed.txt       - Mixed load test"
echo "  📁 profiles/optimized_cpu.prof       - CPU profile"
echo "  📁 profiles/optimized_heap.prof      - Heap profile"
echo "  📁 profiles/optimized_goroutine.prof - Goroutine profile"
echo "  📁 results/optimized_db_metrics.json - DB metrics"
echo ""
echo "For detailed comparison:"
echo "  benchstat benchmarks/baseline.txt benchmarks/optimized.txt"
echo "  go tool pprof -http=:8081 profiles/optimized_cpu.prof"
echo "  diff results/baseline_create.txt results/optimized_create.txt"
echo "════════════════════════════════════════════════════"
