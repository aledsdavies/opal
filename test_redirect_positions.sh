#!/bin/bash
# Comprehensive test of redirect positions with all operator combinations
# Tests WHERE the redirect appears in relation to operators

set -e
TESTDIR="/tmp/opal_redirect_pos_$$"
mkdir -p "$TESTDIR"
cd "$TESTDIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

test_count=0

run_test() {
    local name="$1"
    local cmd="$2"
    
    test_count=$((test_count + 1))
    echo -e "${BLUE}Test $test_count: $name${NC}"
    echo "  Command: $cmd"
    
    # Clean up
    rm -f out.txt out1.txt out2.txt
    
    # Run command
    set +e
    stdout=$(bash -c "$cmd" 2>/dev/null)
    exit_code=$?
    set -e
    
    # Check files
    file_out=""
    file_out1=""
    file_out2=""
    if [ -f out.txt ]; then
        file_out=$(cat out.txt)
    fi
    if [ -f out1.txt ]; then
        file_out1=$(cat out1.txt)
    fi
    if [ -f out2.txt ]; then
        file_out2=$(cat out2.txt)
    fi
    
    echo "  Results:"
    echo "    stdout:   '$stdout'"
    if [ -n "$file_out" ]; then
        echo "    out.txt:  '$file_out'"
    fi
    if [ -n "$file_out1" ]; then
        echo "    out1.txt: '$file_out1'"
    fi
    if [ -n "$file_out2" ]; then
        echo "    out2.txt: '$file_out2'"
    fi
    echo "    exit:     $exit_code"
    echo
}

echo "=== Comprehensive Redirect Position Tests ==="
echo

# ============================================================================
# Category 1: Redirect on FIRST command with operators
# ============================================================================
echo "=== Category 1: Redirect on FIRST command ==="
echo

run_test \
    "first > && second" \
    'echo "a" > out.txt && echo "b"'

run_test \
    "first >> && second" \
    'echo "a" >> out.txt && echo "b"'

run_test \
    "first > || second" \
    'echo "a" > out.txt || echo "b"'

run_test \
    "first > | second" \
    'echo "a" > out.txt | cat'

run_test \
    "first > ; second" \
    'echo "a" > out.txt; echo "b"'

# ============================================================================
# Category 2: Redirect on SECOND command with operators
# ============================================================================
echo "=== Category 2: Redirect on SECOND command ==="
echo

run_test \
    "first && second >" \
    'echo "a" && echo "b" > out.txt'

run_test \
    "first && second >>" \
    'echo "a" && echo "b" >> out.txt'

run_test \
    "first || second >" \
    'echo "a" || echo "b" > out.txt'

run_test \
    "first | second >" \
    'echo "a" | cat > out.txt'

run_test \
    "first ; second >" \
    'echo "a"; echo "b" > out.txt'

# ============================================================================
# Category 3: Redirect on BOTH commands
# ============================================================================
echo "=== Category 3: Redirect on BOTH commands ==="
echo

run_test \
    "first > && second >" \
    'echo "a" > out1.txt && echo "b" > out2.txt'

run_test \
    "first > || second >" \
    'echo "a" > out1.txt || echo "b" > out2.txt'

run_test \
    "first > | second >" \
    'echo "a" > out1.txt | cat > out2.txt'

run_test \
    "first > ; second >" \
    'echo "a" > out1.txt; echo "b" > out2.txt'

# ============================================================================
# Category 4: Redirect to SAME file (overwrite behavior)
# ============================================================================
echo "=== Category 4: Redirect to SAME file ==="
echo

run_test \
    "first > same && second > same" \
    'echo "a" > out.txt && echo "b" > out.txt'

run_test \
    "first > same && second >> same" \
    'echo "a" > out.txt && echo "b" >> out.txt'

run_test \
    "first >> same && second >> same" \
    'echo "a" >> out.txt && echo "b" >> out.txt'

# ============================================================================
# Category 5: Three commands with redirects
# ============================================================================
echo "=== Category 5: Three commands with redirects ==="
echo

run_test \
    "first > && second && third" \
    'echo "a" > out.txt && echo "b" && echo "c"'

run_test \
    "first && second > && third" \
    'echo "a" && echo "b" > out.txt && echo "c"'

run_test \
    "first && second && third >" \
    'echo "a" && echo "b" && echo "c" > out.txt'

run_test \
    "first > && second > && third >" \
    'echo "a" > out1.txt && echo "b" > out.txt && echo "c" > out2.txt'

# ============================================================================
# Category 6: Mixed operators with redirects
# ============================================================================
echo "=== Category 6: Mixed operators with redirects ==="
echo

run_test \
    "first > && second | third" \
    'echo "a" > out.txt && echo "b" | cat'

run_test \
    "first | second > && third" \
    'echo "a" | cat > out.txt && echo "b"'

run_test \
    "first > | second && third" \
    'echo "a" > out.txt | cat && echo "b"'

run_test \
    "first > ; second | third" \
    'echo "a" > out.txt; echo "b" | cat'

run_test \
    "first | second > ; third" \
    'echo "a" | cat > out.txt; echo "b"'

# ============================================================================
# Category 7: Redirect in middle of pipeline
# ============================================================================
echo "=== Category 7: Redirect in middle of pipeline ==="
echo

run_test \
    "first | second > | third" \
    'echo "a" | cat > out.txt | cat'

run_test \
    "first > | second | third" \
    'echo "a" > out.txt | cat | cat'

run_test \
    "first | second | third >" \
    'echo "a" | cat | cat > out.txt'

# ============================================================================
# Category 8: Complex real-world scenarios
# ============================================================================
echo "=== Category 8: Complex real-world scenarios ==="
echo

run_test \
    "build > log && test > results && deploy" \
    'echo "build" > out1.txt && echo "test" > out2.txt && echo "deploy"'

run_test \
    "cmd1 | cmd2 > out && cmd3 || cmd4" \
    'echo "a" | cat > out.txt && echo "b" || echo "c"'

run_test \
    "cmd1 > out1 ; cmd2 | cmd3 > out2 ; cmd4" \
    'echo "a" > out1.txt; echo "b" | cat > out2.txt; echo "c"'

# ============================================================================
# Summary
# ============================================================================
echo "=== Summary ==="
echo "Total tests: $test_count"
echo
echo "Key findings:"
echo "1. Redirects attach to the command they follow"
echo "2. Operators (&&, ||, |, ;) connect commands, not redirects"
echo "3. Multiple redirects to same file: last write wins"
echo "4. Redirect in pipeline: only that command's output is redirected"
echo "5. Exit codes from redirected commands affect && and ||"

# Cleanup
cd /
rm -rf "$TESTDIR"

exit 0
