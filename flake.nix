{
  description = "spotctl: Spotify Web API CLI (refresh-token OAuth) + OpenClaw plugin";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    let
      perSystem = flake-utils.lib.eachDefaultSystem (system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          packages.default = pkgs.buildGoModule {
            pname = "spotctl";
            version = "0.1.0";

            src = ./.;
            subPackages = [ "cmd/spotctl" ];

            vendorHash = null;

            meta = with pkgs.lib; {
              description = "Spotify Web API CLI (refresh-token OAuth)";
              homepage = "https://github.com/joshp123/spotctl";
              license = licenses.mit;
              platforms = platforms.unix;
              mainProgram = "spotctl";
            };
          };

          apps.default = flake-utils.lib.mkApp {
            drv = self.packages.${system}.default;
          };

          devShells.default = pkgs.mkShell {
            packages = with pkgs; [
              go
              gopls
            ];
          };
        }
      );
    in
    perSystem // {
      openclawPlugin = system: {
        name = "spotify";
        skills = [ ./skills/spotify ];
        packages = [ self.packages.${system}.default ];
        needs = {
          stateDirs = [ ];
          requiredEnv = [
            "SPOTIFY_CLIENT_ID"
            "SPOTIFY_CLIENT_SECRET"
            "SPOTIFY_REFRESH_TOKEN"
          ];
        };
      };
    };
}
