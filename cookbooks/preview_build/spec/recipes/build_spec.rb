require 'chefspec'
require 'chefspec/berkshelf'
ChefSpec::Coverage.start!

platforms = {
  'centos' => ['5.9', '6.5']
}

describe 'preview_build::default' do

  platforms.each do |platform_name, platform_versions|

    platform_versions.each do |platform_version|

      context "on #{platform_name} #{platform_version}" do

        let(:chef_run) do
          ChefSpec::Runner.new(platform: platform_name, version: platform_version).converge('preview_build::default')
        end

        before do
          stub_command("/usr/local/go/bin/go version | grep \"go1.2 \"").and_return("1.2")
        end

        it 'includes dependent receipes' do
          expect(chef_run).to include_recipe('golang::default')
          expect(chef_run).to include_recipe('golang::packages')
        end

      end

    end

  end

end
