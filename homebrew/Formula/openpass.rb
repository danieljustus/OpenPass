class Openpass < Formula
  desc "Modern CLI password manager with age encryption"
  homepage "https://github.com/danieljustus/OpenPass"
  url "https://github.com/danieljustus/OpenPass.git", tag: "v1.0.0"
  version "1.0.0"
  license "MIT"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X main.version=#{version}
    ]

    system "go", "build", *std_go_args(ldflags: ldflags), "."
    generate_completions_from_executable(bin/"openpass", "completion")

    man_dir = buildpath/"docs/man"
    system bin/"openpass", "generate", "manpages", man_dir
    man1.install Dir[man_dir/"*.1"]
  end

  def caveats
    <<~EOS
      OpenPass has been installed!
      
      To get started:
        openpass init                    # Initialize a new vault
        openpass set github.com/username # Add your first entry
        openpass get github.com/username # Retrieve it
      
      For MCP server setup:
        openpass mcp-config <name>
      
      Documentation: https://github.com/danieljustus/OpenPass#readme
    EOS
  end

  test do
    system "#{bin}/openpass", "version"
  end
end
