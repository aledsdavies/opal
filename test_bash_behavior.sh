#!/bin/bash
# Comprehensive test of bash behavior for redirect + operator combinations
# Tests all permutations to identify where Opal differs from bash

set -e
TESTDIR="/tmp/opal_bash_test_$$"
mkdir -p "$TESTDIR"
cd "$TESTDIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

test_count=0
pass_count=0

run_test() {
    local name="$1"
    local cmd="$2"
    local expected_stdout="$3"
    local expected_file="$4"
    local expected_exit="$5"
    
    test_count=$((test_count + 1))
    echo -e "${BLUE}Test $test_count: $name${NC}"
    echo "  Command: $cmd"
    
    # Clean up
    rm -f out.txt out1.txt out2.txt
    
    # Run command
    set +e
    actual_stdout=$(bash -c "$cmd" 2>&1)
    actual_exit=$?
    set -e
    
    # Check file if expected
    actual_file=""
    if [ -n "$expected_file" ]; then
        if [ -f out.txt ]; then
            actual_file=$(cat out.txt)
        elif [ -f out1.txt ]; then
            actual_file=$(cat out1.txt)
        fi
    fi
    
    # Compare results
    local pass=true
    if [ "$actual_stdout" != "$expected_stdout" ]; then
        echo -e "  ${RED}FAIL${NC}: stdout mismatch"
        echo "    Expected: '$expected_stdout'"
        echo "    Got:      '$actual_stdout'"
        pass=false
    fi
    
    if [ -n "$expected_file" ] && [ "$actual_file" != "$expected_file" ]; then
        echo -e "  ${RED}FAIL${NC}: file content mismatch"
        echo "    Expected: '$expected_file'"
        echo "    Got:      '$actual_file'"
        pass=false
    fi
    
    if [ "$actual_exit" != "$expected_exit" ]; then
        echo -e "  ${RED}FAIL${NC}: exit code mismatch"
        echo "    Expected: $expected_exit"
        echo "    Got:      $actual_exit"
        pass=false
    fi
    
    if $pass; then
        echo -e "  ${GREEN}PASS${NC}"
        pass_count=$((pass_count + 1))
    fi
    echo
}

echo "=== Comprehensive Bash Redirect + Operator Behavior Tests ==="
echo

# ============================================================================
# Category 1: Redirect (>) with chaining operators
# ============================================================================
echo "=== Category 1: Redirect (>) + Chaining Operators ==="
echo

run_test \
    "redirect > then &&" \
    'echo "a" > out.txt && echo "b"' \
    "b" \
    "a" \
    0

run_test \
    "redirect > then || (success)" \
    'echo "a" > out.txt || echo "b"' \
    "" \
    "a" \
    0

run_test \
    "redirect > then || (failure)" \
    'echo "a" > /nonexistent/out.txt 2>/dev/null || echo "b"' \
    "b" \
    "" \
    0

run_test \
    "redirect > then |" \
    'echo "a" > out.txt | cat' \
    "" \
    "a" \
    0

run_test \
    "redirect > then ;" \
    'echo "a" > out.txt; echo "b"' \
    "b" \
    "a" \
    0

# ============================================================================
# Category 2: Redirect (>>) with chaining operators
# ============================================================================
echo "=== Category 2: Redirect (>>) + Chaining Operators ==="
echo

# Setup initial file
echo "initial" > out.txt

run_test \
    "redirect >> then &&" \
    'echo "a" >> out.txt && echo "b"' \
    "b" \
    "initial
a" \
    0

rm -f out.txt
echo "initial" > out.txt

run_test \
    "redirect >> then ||" \
    'echo "a" >> out.txt || echo "b"' \
    "" \
    "initial
a" \
    0

rm -f out.txt
echo "initial" > out.txt

run_test \
    "redirect >> then |" \
    'echo "a" >> out.txt | cat' \
    "" \
    "initial
a" \
    0

rm -f out.txt
echo "initial" > out.txt

run_test \
    "redirect >> then ;" \
    'echo "a" >> out.txt; echo "b"' \
    "b" \
    "initial
a" \
    0

# ============================================================================
# Category 3: Pipe (|) with redirect
# ============================================================================
echo "=== Category 3: Pipe (|) + Redirect ==="
echo

run_test \
    "pipe | then redirect >" \
    'echo "a" | cat > out.txt' \
    "" \
    "a" \
    0

run_test \
    "pipe | then redirect >>" \
    'echo "a" | cat >> out.txt' \
    "" \
    "a" \
    0

run_test \
    "redirect > then pipe | then redirect >" \
    'echo "a" > out1.txt | cat > out.txt' \
    "" \
    "" \
    0

# ============================================================================
# Category 4: Multiple operators in sequence
# ============================================================================
echo "=== Category 4: Multiple Operators ==="
echo

run_test \
    "redirect > && redirect >" \
    'echo "a" > out1.txt && echo "b" > out.txt' \
    "" \
    "b" \
    0

run_test \
    "redirect > && redirect > && echo" \
    'echo "a" > out1.txt && echo "b" > out.txt && echo "c"' \
    "c" \
    "b" \
    0

run_test \
    "redirect > ; redirect >" \
    'echo "a" > out1.txt; echo "b" > out.txt' \
    "" \
    "b" \
    0

run_test \
    "redirect > | cat && echo" \
    'echo "a" > out.txt | cat && echo "b"' \
    "b" \
    "a" \
    0

run_test \
    "redirect > | cat || echo" \
    'echo "a" > out.txt | cat || echo "b"' \
    "" \
    "a" \
    0

run_test \
    "echo | redirect > && echo" \
    'echo "a" | cat > out.txt && echo "b"' \
    "b" \
    "a" \
    0

# ============================================================================
# Category 5: Chaining operators without redirect (baseline)
# ============================================================================
echo "=== Category 5: Chaining Operators (No Redirect - Baseline) ==="
echo

run_test \
    "echo && echo" \
    'echo "a" && echo "b"' \
    "a
b" \
    "" \
    0

run_test \
    "echo || echo (success)" \
    'echo "a" || echo "b"' \
    "a" \
    "" \
    0

run_test \
    "false || echo" \
    'false || echo "b"' \
    "b" \
    "" \
    0

run_test \
    "echo | cat" \
    'echo "a" | cat' \
    "a" \
    "" \
    0

run_test \
    "echo ; echo" \
    'echo "a"; echo "b"' \
    "a
b" \
    "" \
    0

run_test \
    "echo | cat | wc -l" \
    'echo "a" | cat | wc -l' \
    "1" \
    "" \
    0

# ============================================================================
# Category 6: Complex combinations
# ============================================================================
echo "=== Category 6: Complex Combinations ==="
echo

run_test \
    "redirect > && pipe |" \
    'echo "a" > out.txt && echo "b" | cat' \
    "b" \
    "a" \
    0

run_test \
    "pipe | && redirect >" \
    'echo "a" | cat && echo "b" > out.txt' \
    "" \
    "b" \
    0

run_test \
    "redirect > ; pipe | ; redirect >" \
    'echo "a" > out1.txt; echo "b" | cat; echo "c" > out.txt' \
    "b" \
    "c" \
    0

run_test \
    "three pipes with middle redirect" \
    'echo "a" | cat | cat > out.txt | cat' \
    "" \
    "a" \
    0

# ============================================================================
# Category 7: Edge cases
# ============================================================================
echo "=== Category 7: Edge Cases ==="
echo

run_test \
    "empty command with redirect" \
    ': > out.txt && echo "b"' \
    "b" \
    "" \
    0

run_test \
    "redirect to same file twice" \
    'echo "a" > out.txt && echo "b" > out.txt' \
    "" \
    "b" \
    0

run_test \
    "redirect with variable in path" \
    'FILE=out.txt; echo "a" > $FILE && cat $FILE' \
    "a" \
    "a" \
    0

# ============================================================================
# Summary
# ============================================================================
echo "=== Summary ==="
echo "Tests passed: $pass_count / $test_count"
echo

if [ $pass_count -eq $test_count ]; then
    echo -e "${GREEN}All tests passed!${NC}"
else
    echo -e "${RED}Some tests failed.${NC}"
fi

# Cleanup
cd /
rm -rf "$TESTDIR"

exit 0
