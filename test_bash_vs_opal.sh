#!/bin/bash
# Compare bash vs Opal behavior for redirect + operator combinations

TESTDIR="/tmp/opal_comparison_$$"
mkdir -p "$TESTDIR"
cd "$TESTDIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

test_count=0
match_count=0

compare_test() {
    local name="$1"
    local cmd="$2"
    
    test_count=$((test_count + 1))
    echo -e "${BLUE}Test $test_count: $name${NC}"
    echo "  Command: $cmd"
    
    # Clean up
    rm -f out.txt out1.txt out2.txt
    
    # Run in bash
    set +e
    bash_stdout=$(bash -c "$cmd" 2>/dev/null)
    bash_exit=$?
    bash_file=""
    if [ -f out.txt ]; then
        bash_file=$(cat out.txt)
    fi
    set -e
    
    # Run in Opal (if available)
    rm -f out.txt out1.txt out2.txt
    if command -v opal &> /dev/null; then
        set +e
        opal_stdout=$(echo "$cmd" | opal 2>/dev/null)
        opal_exit=$?
        opal_file=""
        if [ -f out.txt ]; then
            opal_file=$(cat out.txt)
        fi
        set -e
        
        # Compare
        local match=true
        if [ "$bash_stdout" != "$opal_stdout" ]; then
            echo -e "  ${RED}DIFF${NC}: stdout mismatch"
            echo "    Bash:  '$bash_stdout'"
            echo "    Opal:  '$opal_stdout'"
            match=false
        fi
        
        if [ "$bash_file" != "$opal_file" ]; then
            echo -e "  ${RED}DIFF${NC}: file content mismatch"
            echo "    Bash:  '$bash_file'"
            echo "    Opal:  '$opal_file'"
            match=false
        fi
        
        if [ "$bash_exit" != "$opal_exit" ]; then
            echo -e "  ${RED}DIFF${NC}: exit code mismatch"
            echo "    Bash:  $bash_exit"
            echo "    Opal:  $opal_exit"
            match=false
        fi
        
        if $match; then
            echo -e "  ${GREEN}MATCH${NC}"
            match_count=$((match_count + 1))
        fi
    else
        echo -e "  ${YELLOW}SKIP${NC}: opal not found"
        echo "    Bash stdout: '$bash_stdout'"
        echo "    Bash file:   '$bash_file'"
        echo "    Bash exit:   $bash_exit"
    fi
    echo
}

echo "=== Bash vs Opal Behavior Comparison ==="
echo

# Critical tests for redirect + chaining operators
compare_test "redirect > then &&" 'echo "a" > out.txt && echo "b"'
compare_test "redirect > then ||" 'echo "a" > out.txt || echo "b"'
compare_test "redirect > then |" 'echo "a" > out.txt | cat'
compare_test "redirect > then ;" 'echo "a" > out.txt; echo "b"'
compare_test "redirect >> then &&" 'echo "a" >> out.txt && echo "b"'
compare_test "pipe | then redirect >" 'echo "a" | cat > out.txt'
compare_test "redirect > && redirect >" 'echo "a" > out1.txt && echo "b" > out.txt'
compare_test "redirect > | cat && echo" 'echo "a" > out.txt | cat && echo "b"'

echo "=== Summary ==="
echo "Tests matched: $match_count / $test_count"

cd /
rm -rf "$TESTDIR"
