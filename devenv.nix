{ pkgs, ... }:
{
  packages = with pkgs; [
    go
    gopls
    jq
  ];

  scripts.test.exec = ''
    go test ./...
  '';

  scripts.build.exec = ''
    mkdir -p .out
    go build -o .out/spotctl ./cmd/spotctl
    echo "built: .out/spotctl"
  '';

  scripts.lint.exec = ''
    gofmt -w .
    go test ./...
  '';

  scripts.smoke.exec = ''
    ./scripts/smoke.sh
  '';

  enterShell = ''
    echo "devenv: try: devenv shell -- test | devenv shell -- build | devenv shell -- smoke"
  '';
}
