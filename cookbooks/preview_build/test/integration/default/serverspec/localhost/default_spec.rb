require 'spec_helper'

describe command('/usr/local/go/bin/go version') do
  its(:stdout) { should match /go version go1\.2/ }
end
