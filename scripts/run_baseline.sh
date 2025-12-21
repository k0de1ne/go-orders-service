#!/bin/bash
set -e

echo "🚀 Starting baseline performance testing..."
echo ""

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
go test -bench=. -benchmem ./... | tee benchmarks/baseline.txt
echo ""

echo "⚡ Load test: Create (1000 requests, 10 concurrent)..."
go run scripts/load_test.go \
    -url http://localhost:8080 \
    -requests 1000 \
    -concurrency 10 \
    -operation create \
    | tee results/baseline_create.txt
echo ""

echo "⚡ Load test: Get (2000 requests, 20 concurrent)..."
go run scripts/load_test.go \
    -url http://localhost:8080 \
    -requests 2000 \
    -concurrency 20 \
    -operation get \
    | tee results/baseline_get.txt
echo ""

echo "⚡ Load test: Mixed (1000 requests, 15 concurrent)..."
go run scripts/load_test.go \
    -url http://localhost:8080 \
    -requests 1000 \
    -concurrency 15 \
    -operation mixed \
    | tee results/baseline_mixed.txt
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
curl -s http://localhost:6060/debug/pprof/profile?seconds=30 > profiles/baseline_cpu.prof
wait $LOAD_PID
echo "✅ CPU profile saved: profiles/baseline_cpu.prof"
echo ""

echo "🔍 Collecting Heap profile..."
curl -s http://localhost:6060/debug/pprof/heap > profiles/baseline_heap.prof
echo "✅ Heap profile saved: profiles/baseline_heap.prof"
echo ""

echo "🔍 Collecting Goroutine profile..."
curl -s http://localhost:6060/debug/pprof/goroutine > profiles/baseline_goroutine.prof
echo "✅ Goroutine profile saved: profiles/baseline_goroutine.prof"
echo ""

echo "📊 Collecting Database Pool metrics..."
curl -s http://localhost:8080/metrics/db | jq '.' | tee results/baseline_db_metrics.json
echo ""

echo "════════════════════════════════════════════════════"
echo "✅ Baseline testing completed!"
echo ""
echo "Results saved in:"
echo "  📁 benchmarks/baseline.txt          - Go benchmarks"
echo "  📁 results/baseline_create.txt      - Create load test"
echo "  📁 results/baseline_get.txt         - Get load test"
echo "  📁 results/baseline_mixed.txt       - Mixed load test"
echo "  📁 profiles/baseline_cpu.prof       - CPU profile"
echo "  📁 profiles/baseline_heap.prof      - Heap profile"
echo "  📁 profiles/baseline_goroutine.prof - Goroutine profile"
echo "  📁 results/baseline_db_metrics.json - DB metrics"
echo ""
echo "Next steps:"
echo "  1. Apply optimizations to the code"
echo "  2. Run: ./scripts/run_optimized.sh"
echo "  3. Compare results: benchstat benchmarks/baseline.txt benchmarks/optimized.txt"
echo "════════════════════════════════════════════════════"
