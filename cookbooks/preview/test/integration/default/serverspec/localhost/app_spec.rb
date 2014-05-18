require 'spec_helper'

describe 'preview app' do

  describe user('preview') do
    it { should exist }
  end

  describe group('preview') do
    it { should exist }
  end

  describe file('/home/preview/preview') do
    it { should be_file }
    it { should be_owned_by 'preview' }
    it { should be_grouped_into 'preview' }
    it { should be_executable }
  end

  describe file('/etc/preview.conf') do
    it { should be_file }
    it { should be_owned_by 'preview' }
    it { should be_grouped_into 'preview' }
    it { should contain('/home/preview/data/') }
  end

  describe command('curl -v http://localhost:8080/admin/config') do
    its(:stdout) { should match /HTTP\/1.1 200 OK/ }
    its(:stdout) { should match /\/home\/preview\/data\// }
  end

  describe command('convert --version') do
    its(:stdout) { should match /(6.2.8|6.5.4)/ }
  end

  describe command('libreoffice4.2 --version') do
    its(:stdout) { should match /4.2/ }
  end

  describe command('pdfinfo -v') do
    its(:stdout) { should match /(3.00|0.12.4)/ }
  end

end
