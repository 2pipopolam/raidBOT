{ pkgs, lib, config, inputs, ... }:

{
  # Enable Go language support
  languages.go.enable = true;
  
  # Required packages for the bot
  packages = with pkgs; [ 
    git
    yt-dlp          # For YouTube audio extraction
    ffmpeg          # For audio conversion
    #opus            # For audio encoding
    pkg-config      # Required for building Go dependencies
    gcc             # Required for building Go dependencies
    golangci-lint   # Нужен для pre-commit хуков
  ];

  # Set up environment variables for the bot
  env = {
    CGO_ENABLED = "1";
    #PKG_CONFIG_PATH = "${pkgs.opus}/lib/pkgconfig";
    CGO_CFLAGS = "-w";  # Отключаем предупреждения при компиляции C кода
  };
  
  # Process definition for devenv up
  processes.bot.exec = "go run .";

  # Pre-commit hooks configuration
  pre-commit.hooks = {
    gofmt.enable = true;
    golangci-lint = {
      enable = true;
      package = pkgs.golangci-lint;
    };
  };

  # Add shell environment for Go tools
  enterShell = ''
    export PATH=$PATH:${pkgs.go}/bin
    export PATH=$PATH:${pkgs.golangci-lint}/bin
  '';
}
