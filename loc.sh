#!/bin/bash
echo "=== Code Statistics ==="
echo ""

# Total Go LOC (excluding vendor)
total=$(find . -name "*.go" ! -path "*/vendor/*" -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
echo "Total Go code: $total lines"
echo ""

# Production vs Test
prod=$(find . -name "*.go" ! -path "*/vendor/*" ! -name "*_test.go" -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
test=$(find . -name "*_test.go" -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
echo "Production code: $prod lines"
echo "Test code: $test lines"
if [ "$prod" -gt 0 ]; then
    ratio=$(awk "BEGIN {printf \"%.2f:1\", $test/$prod}")
    echo "Test ratio: $ratio"
fi
echo ""

# Top 10 largest Go files
echo "Top 10 largest files:"
find . -name "*.go" ! -path "*/vendor/*" -type f -exec wc -l {} + | sort -rn | head -10 | while read lines file; do
    echo "  $lines  $file"
done
