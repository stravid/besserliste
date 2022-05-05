{ pkgs ? import (fetchGit {
  url = https://github.com/NixOS/nixpkgs;
  ref = "d5d94127fd6468febe4f5e8eba8cb231bbd56103";
}) {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go_1_18
    watchexec
    ruby
    graphviz
    icu
    tokei
  ];

  shellHook = ''
    mkdir -p .local-data/gems
    export GEM_HOME=$PWD/.local-data/gems
    export GEM_PATH=$GEM_HOME
    export PATH="$GEM_PATH/bin:$PATH"
    gem install erde
  '';
}
