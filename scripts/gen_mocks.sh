#!/bin/bash
set -e

MOCKS_DIR="internal/mocks"

mkdir -p "$MOCKS_DIR"

echo "scanning interfaces in internal/..."

find "internal/" -name "*.go" \
    -not -path "*/mocks/*" \
    -not -path "*/vendor/*" | while read -r file; do

    if ! grep -q "interface" "$file"; then
        continue
    fi

    rel="${file#internal/}"
    base="${rel//\//_}"
    base="${base%.go}"
    output_file="$MOCKS_DIR/${base}_mock.go"

    echo "SUCCESS: $file → $output_file"

    mockgen \
        -source="$file" \
        -destination="$output_file" \
        -package="mocks" \
        2>/dev/null || {
            echo "WARNING: no interfaces found in $file"
        }
done

echo "all mocks generated in $MOCKS_DIR/"