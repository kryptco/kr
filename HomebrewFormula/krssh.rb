require "language/go"
class Krssh < Formula
  desc ""
  homepage ""
  url "https://bitbucket.org/kryptco/krssh/get/master.tar.gz"
  version "0.1.0"
  sha256 "593808489cd16487b2d2a05f561231167e8d776b907d6c43e6dbaf7ef970cef3"
  head "https://bitbucket.org/kryptco/krssh.git"

  bottle do
	# TODO: add bottle URL since its non-standard
    cellar :any_skip_relocation
    sha256 "64db158ba7e9356c9acf2d307d2bec7b5d8547ea54b38937a1427f9ffd361f74" => :el_capitan
  end

  def install
    # ENV.deparallelize  # if your formula fails when building in parallel

    # Remove unrecognized options if warned by configure
    system "make", "install" # if this fails, try separate make/make install steps
  end

  depends_on "go" => :build
  #go_resource "bitbucket.org/kryptco/qr/coding" do
	#url "https://bitbucket.org/kryptco/qr.git",
		#:revision => "b8c65851ea3cca1ce89eb2dce17d0d924196a02b"
  #end

  #go_resource "github.com/urfave/cli" do
	#url "https://github.com/urfave/cli.git",
		#:revision => "a14d7d367bc02b1f57d88de97926727f2d936387"
  #end

  def install
	  ENV["GOPATH"] = buildpath
	  ENV["GOOS"] = "darwin"
	  ENV["GOARCH"] = MacOS.prefer_64_bit? ? "amd64" : "386"

	  dir = buildpath/"src/bitbucket.org/kryptco/krssh"
	  dir.install buildpath.children

	  #mkdir_p buildpath/"src/bitbucket.org/kryptco"
	  #ln_s buildpath, buildpath/"src/bitbucket.org/kryptco/krssh"
	  #Language::Go.stage_deps resources, buildpath/"src"

	  cd "src/bitbucket.org/kryptco/krssh/ctl" do
		  system "go", "build", "-o", bin/"kr"
	  end
	  cd "src/bitbucket.org/kryptco/krssh/agent" do
		  system "go", "build", "-o", bin/"krssh-agent"
	  end
  end

  plist_options :startup => "true"

  def plist; <<-EOS.undent
	  <?xml version="1.0" encoding="UTF-8"?>
	  <!DOCTYPE plist PUBLIC "-//Apple Computer//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
		<plist version="1.0">
		<dict>
			<key>Label</key>
			<string>#{plist_name}</string>
			<key>ProgramArguments</key>
			<array>
				<string>/usr/local/bin/krssh-agent</string>
			</array>
			<key>Sockets</key>
			<dict>
				<key>AuthListener</key>
				<dict>
					<key>SecureSocketWithKey</key>
					<string>KRSSH_AUTH_SOCK</string>
				</dict>
				<key>CtlListener</key>
				<dict>
					<key>SecureSocketWithKey</key>
					<string>KRSSH_CTL_SOCK</string>
				</dict>
			</dict>
			<key>EnableTransactions</key>
			<true/>
		</dict>
		</plist>
    EOS
  end

   def caveats; <<-EOS.undent
	   You're almost there! Point SSH to the kryptonite ssh-agent by running this
	   command and restarting your terminal:

	      echo "export SSH_AUTH_SOCK=\\$KRSSH_AUTH_SOCK" >> #{shell_profile}
  EOS
  end

end
