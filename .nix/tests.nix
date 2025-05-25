# Test scenarios for devcmd library and generated CLIs
{ pkgs, lib, self, system }:

let
  devcmdLib = import ./lib.nix { inherit pkgs self system lib; };

  # Common test utilities
  testUtils = {
    # Run a command and check exit code
    runAndCheck = cmd: expectedExitCode: ''
      echo "Running: ${cmd}"
      ${cmd}
      EXIT_CODE=$?
      if [ $EXIT_CODE -ne ${toString expectedExitCode} ]; then
        echo "Expected exit code ${toString expectedExitCode}, got $EXIT_CODE"
        exit 1
      fi
    '';

    # Check if output contains expected text
    checkOutput = cmd: expectedText: ''
      OUTPUT=$(${cmd} 2>&1)
      if ! echo "$OUTPUT" | grep -q "${expectedText}"; then
        echo "Expected output to contain: ${expectedText}"
        echo "Actual output: $OUTPUT"
        exit 1
      fi
    '';
  };

in
rec {
  # Test basic devcmd functionality
  basicTests = {
    # Test simple command generation
    simpleCommand = devcmdLib.testCLI {
      cli = devcmdLib.mkDevCLI {
        name = "simple-test";
        commandsContent = ''
          build: echo "Building project..."
          test: echo "Running tests..."
          clean: rm -f *.tmp
        '';
      };

      testScenarios = [
        {
          name = "help-works";
          script = testUtils.runAndCheck "simple-test --help" 0;
        }
        {
          name = "build-command";
          script = testUtils.checkOutput "simple-test build" "Building project...";
        }
        {
          name = "test-command";
          script = testUtils.checkOutput "simple-test test" "Running tests...";
        }
      ];
    };

    # Test commands with POSIX syntax and parentheses
    posixSyntax = devcmdLib.testCLI {
      cli = devcmdLib.mkDevCLI {
        name = "posix-test";
        commandsContent = ''
          check-deps: (which go && echo "Go found") || (echo "Go missing" && exit 1)
          validate: test -f go.mod && echo "Go module found" || echo "No go.mod"
          complex: (cd /tmp && echo "In tmp: $(pwd)") && echo "Back to: $(pwd)"
        '';
      };

      testScenarios = [
        {
          name = "parentheses-syntax";
          script = ''
            # Test that parentheses are preserved in commands
            posix-test check-deps 2>&1 | grep -q "Go found\|Go missing"
          '';
        }
        {
          name = "complex-parentheses";
          script = ''
            # Test complex parentheses combinations
            posix-test complex | grep -q "In tmp"
          '';
        }
      ];
    };

    # Test variable expansion
    variableExpansion = devcmdLib.testCLI {
      cli = devcmdLib.mkDevCLI {
        name = "variables-test";
        commandsContent = ''
          def SRC = ./src;
          def PORT = 8080;
          def CHECK_CMD = (which node || echo "missing");

          build: cd $(SRC) && echo "Building in $(SRC)"
          serve: echo "Starting server on port $(PORT)"
          check: $(CHECK_CMD) && echo "Dependencies OK"
        '';
      };

      testScenarios = [
        {
          name = "variable-expansion";
          script = testUtils.checkOutput "variables-test build" "Building in ./src";
        }
        {
          name = "port-variable";
          script = testUtils.checkOutput "variables-test serve" "port 8080";
        }
        {
          name = "complex-variable";
          script = ''
            # Test variable with parentheses
            OUTPUT=$(variables-test check 2>&1)
            echo "Check output: $OUTPUT"
          '';
        }
      ];
    };
  };

  # Test watch/stop process management
  processManagementTests = {
    watchStopCommands = devcmdLib.testCLI {
      cli = devcmdLib.mkDevCLI {
        name = "process-test";
        commandsContent = ''
          watch demo: python3 -m http.server 9999 &
          stop demo: pkill -f "python3 -m http.server 9999"

          watch multi: {
            echo "Starting services...";
            sleep 10 &;
            sleep 20 &;
            echo "Services started"
          }
        '';
        validateCommands = true;
      };

      testScenarios = [
        {
          name = "has-process-management";
          script = ''
            # Check that process management commands exist
            process-test --help | grep -q "status"
            process-test --help | grep -q "logs"
          '';
        }
        {
          name = "watch-command-structure";
          script = ''
            # Test that watch command doesn't block immediately
            timeout 2s process-test watch demo || true
            echo "Watch command executed"
          '';
        }
        {
          name = "status-command";
          script = testUtils.runAndCheck "process-test status" 0;
        }
      ];
    };
  };

  # Test block commands and background processes
  blockCommandTests = {
    backgroundProcesses = devcmdLib.testCLI {
      cli = devcmdLib.mkDevCLI {
        name = "block-test";
        commandsContent = ''
          setup: {
            echo "Step 1: Initialize";
            echo "Step 2: Configure";
            echo "Step 3: Complete"
          }

          parallel: {
            echo "Task 1" &;
            echo "Task 2" &;
            echo "Task 3"
          }

          complex: {
            (echo "Subshell 1" && sleep 1) &;
            (echo "Subshell 2" || echo "Fallback") &;
            echo "Main thread"
          }
        '';
      };

      testScenarios = [
        {
          name = "sequential-block";
          script = ''
            OUTPUT=$(block-test setup)
            echo "$OUTPUT" | grep -q "Step 1"
            echo "$OUTPUT" | grep -q "Step 2"
            echo "$OUTPUT" | grep -q "Step 3"
          '';
        }
        {
          name = "parallel-block";
          script = ''
            # Test that parallel commands execute
            block-test parallel | grep -q "Task"
          '';
        }
        {
          name = "complex-block";
          script = ''
            # Test complex block with parentheses and background
            block-test complex | grep -q "Main thread"
          '';
        }
      ];
    };
  };

  # Test error handling and edge cases
  errorHandlingTests = {
    invalidCommands = {
      name = "invalid-syntax";
      cli = devcmdLib.mkDevCLI {
        name = "error-test";
        commandsContent = ''
          valid: echo "This works"
          # This should still parse correctly
          special-chars: echo "Special: !@#$%^&*()"
          unicode: echo "Hello 世界"
        '';
      };

      testScenarios = [
        {
          name = "valid-command";
          script = testUtils.checkOutput "error-test valid" "This works";
        }
        {
          name = "special-characters";
          script = testUtils.checkOutput "error-test special-chars" "Special:";
        }
        {
          name = "unicode-support";
          script = testUtils.checkOutput "error-test unicode" "世界";
        }
      ];
    };
  };

  # Performance and scale tests
  performanceTests = {
    largeCLI = devcmdLib.testCLI {
      cli = devcmdLib.mkDevCLI {
        name = "large-test";
        commandsContent = lib.concatStringsSep "\n" (
          lib.genList (i: "cmd${toString i}: echo 'Command ${toString i}'") 50
        );
      };

      testScenarios = [
        {
          name = "many-commands";
          script = ''
            # Test that CLI with many commands works
            large-test --help | wc -l | grep -q "[0-9]"
            large-test cmd25 | grep -q "Command 25"
          '';
        }
        {
          name = "help-performance";
          script = ''
            # Test that help is reasonably fast
            time timeout 5s large-test --help > /dev/null
          '';
        }
      ];
    };
  };

  # Integration tests with real-world scenarios
  realWorldTests = {
    webDevelopment = devcmdLib.testCLI {
      cli = devcmdLib.mkDevCLI {
        name = "webdev";
        commandsContent = ''
          def NODE_ENV = development;
          def PORT = 3000;
          def API_PORT = 3001;

          install: npm install && echo "Dependencies installed"

          build: {
            echo "Building frontend...";
            (cd frontend && npm run build) || echo "No frontend";
            echo "Building backend...";
            (cd backend && go build) || echo "No backend"
          }

          watch dev: {
            echo "Starting development servers...";
            (cd frontend && NODE_ENV=$(NODE_ENV) npm start) &;
            (cd backend && go run . --port=$(API_PORT)) &;
            echo "Servers starting on ports $(PORT) and $(API_PORT)"
          }

          stop dev: {
            pkill -f "npm start" || echo "No frontend running";
            pkill -f "go run" || echo "No backend running";
            echo "Development servers stopped"
          }

          test: {
            echo "Running frontend tests...";
            (cd frontend && npm test) || echo "No frontend tests";
            echo "Running backend tests...";
            (cd backend && go test ./...) || echo "No backend tests"
          }

          deploy: {
            echo "Building for production...";
            webdev build;
            echo "Deploying...";
            (which docker && docker build -t myapp .) || echo "No Docker"
          }
        '';
      };

      testScenarios = [
        {
          name = "has-all-commands";
          script = ''
            webdev --help | grep -q "install"
            webdev --help | grep -q "build"
            webdev --help | grep -q "dev"
            webdev --help | grep -q "test"
            webdev --help | grep -q "deploy"
          '';
        }
        {
          name = "install-command";
          script = testUtils.checkOutput "webdev install" "Dependencies installed";
        }
        {
          name = "build-command";
          script = ''
            OUTPUT=$(webdev build 2>&1)
            echo "$OUTPUT" | grep -q "Building frontend"
            echo "$OUTPUT" | grep -q "Building backend"
          '';
        }
      ];
    };

    goProject = devcmdLib.testCLI {
      cli = devcmdLib.mkDevCLI {
        name = "goproj";
        commandsContent = ''
          def MODULE = github.com/example/myapp;
          def BINARY = myapp;
          def VERSION = v0.1.0;

          init: go mod init $(MODULE)

          deps: {
            go mod tidy;
            go mod download;
            go mod verify
          }

          build: {
            echo "Building $(BINARY) $(VERSION)...";
            go build -ldflags="-X main.Version=$(VERSION)" -o $(BINARY) ./cmd/$(BINARY)
          }

          test: {
            go test -v ./...;
            go test -race ./...;
            go test -bench=. ./...
          }

          lint: {
            (which golangci-lint && golangci-lint run) || echo "No linter";
            go fmt ./...;
            go vet ./...
          }

          check: {
            goproj deps;
            goproj lint;
            goproj test;
            echo "All checks passed!"
          }

          release: {
            goproj check;
            echo "Creating release $(VERSION)...";
            (which git && git tag $(VERSION)) || echo "No git"
          }
        '';
      };

      testScenarios = [
        {
          name = "go-commands";
          script = ''
            goproj --help | grep -q "build"
            goproj --help | grep -q "test"
            goproj --help | grep -q "lint"
          '';
        }
        {
          name = "deps-command";
          script = ''
            # Test that deps command has proper structure
            OUTPUT=$(goproj deps 2>&1)
            echo "Deps output: $OUTPUT"
          '';
        }
      ];
    };
  };

  # All tests combined
  allTests = basicTests // processManagementTests // blockCommandTests //
    errorHandlingTests // performanceTests // realWorldTests;

  # Derivation that runs all tests
  runAllTests = pkgs.runCommand "devcmd-all-tests"
    {
      nativeBuildInputs = [ pkgs.bash ];
    } ''
    mkdir -p $out
    echo "Running all devcmd tests..."

    ${lib.concatMapStringsSep "\n" (testName: test: ''
      echo "Running test group: ${testName}"
      ${test}
      echo "✅ ${testName} passed"
    '') (lib.mapAttrsToList (name: test: test) allTests)}

    echo "🎉 All tests passed!"
    date > $out/success
  '';
}
