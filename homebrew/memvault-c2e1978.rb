class Memvault < Formula
  desc "Local session memory and conversation archive"
  homepage "https://github.com/memvault-org/memvault"
  version "c2e1978"
  license "MIT"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/memvault-org/memvault/releases/download/vc2e1978/memvault_c2e1978_darwin_arm64.tar.gz"
    sha256 "77148f03a91ee65425a524b676731f410a01afacb9e55f86afe9c69bcea28197"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/memvault-org/memvault/releases/download/vc2e1978/memvault_c2e1978_darwin_amd64.tar.gz"
    sha256 "3ef606f9f5f84c599656f62d376b662eb6b539d2c1ff0a630640f2bb27841ada"
  elsif OS.linux? && Hardware::CPU.arm? && Hardware::CPU.is_64_bit?
    url "https://github.com/memvault-org/memvault/releases/download/vc2e1978/memvault_c2e1978_linux_arm64.tar.gz"
    sha256 "99de0bc3f4eade6637075edf0ea6ad6742e8d768381c39dbea83c8fee4f919da"
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/memvault-org/memvault/releases/download/vc2e1978/memvault_c2e1978_linux_amd64.tar.gz"
    sha256 "043546c613813485916f638321239e6ac7b325f41d8d30213b46617ce4662673"
  end

  def install
    bin.install "memvault"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/memvault version")
  end
end
