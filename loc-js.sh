#!/bin/bash
echo "=== JavaScript Code Statistics ==="
echo ""

# Total JavaScript LOC
total=$(find static/js -name "*.js" -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
echo "Total JavaScript code: $total lines"
echo ""

# Production vs Test (test files named with "test")
prod=$(find static/js -name "*.js" ! -name "*test*.js" -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
test=$(find static/js -name "*test*.js" -exec wc -l {} + 2>/dev/null | tail -1 | awk '{print $1}')
echo "Production code: $prod lines"
echo "Test code: $test lines"
if [ "$prod" -gt 0 ]; then
    ratio=$(awk "BEGIN {printf \"%.2f:1\", $test/$prod}")
    echo "Test ratio: $ratio"
fi
echo ""

# All JavaScript files
echo "All JavaScript files:"
find static/js -name "*.js" -type f -exec wc -l {} + | sort -rn | while read lines file; do
    echo "  $lines  $file"
done
