{ pkgs ? import (fetchGit {
  url = https://github.com/NixOS/nixpkgs;
  ref = "9675a865c9c3eeec36c06361f7215e109925654c";
}) {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go_1_17
    watchexec
    ruby
    graphviz
  ];

  shellHook = ''
    mkdir -p .local-data/gems
    export GEM_HOME=$PWD/.local-data/gems
    export GEM_PATH=$GEM_HOME
    export PATH="$GEM_PATH/bin:$PATH"
    gem install erde
  '';
}
