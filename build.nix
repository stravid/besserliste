{ pkgs ? import (fetchGit {
  url = https://github.com/NixOS/nixpkgs;
  ref = "d5d94127fd6468febe4f5e8eba8cb231bbd56103";
}) {} }:

let
  icu = pkgs.icu.overrideAttrs (attrs: {
    dontDisableStatic = true;
    configureFlags = [ "--disable-debug" "--enable-static" "--disable-shared" ];
    outputs = attrs.outputs ++ [ "static" ];
    postInstall = ''
      mkdir -p $static/lib
      mv -v lib/*.a $static/lib
    '' + (attrs.postInstall or "");
  });
in  pkgs.mkShell {
  buildInputs = with pkgs; [
    go_1_18
    glibc
    icu.static
  ];

  shellHook = ''
      echo ${icu.dev};
      echo ${icu};
      echo ${icu.static};
      echo ${pkgs.glibc};
      echo ${pkgs.glibc.static};
      echo ${pkgs.glibc.out};

      #export GOOS=linux;
      #export GOARCH=amd64;
      #export CC=gcc;
      #export CXX=g++;
      #export LD_DEBUG=libs;
      #export LD_DEBUG_OUTPUT=debug;
      #export CGO_ENABLED=1;

      #export CFLAGS="-DSQLITE_ENABLE_ICU";
      #export LDFLAGS="-L${pkgs.glibc}/lib -L${pkgs.icu}/lib";
      export CGO_CFLAGS="-I${pkgs.icu.dev}/include";
      export CGO_LDFLAGS="-L${icu.static}/lib -lstdc++";
      #printenv;
    '';

}
