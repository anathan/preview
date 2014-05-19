require 'chefspec'
require 'chefspec/berkshelf'
ChefSpec::Coverage.start!

platforms = {
  'centos' => ['5.9']
}

describe 'preview::app' do

  platforms.each do |platform_name, platform_versions|

    platform_versions.each do |platform_version|

      context "no install type on #{platform_name} #{platform_version}" do

        let(:chef_run) do
          ChefSpec::Runner.new(platform: platform_name, version: platform_version) do |node|
            node.set['preview']['install_type'] = 'none'
          end.converge('preview::app')
        end

        it 'includes dependent receipes' do
          expect(chef_run).to include_recipe('apt')
          expect(chef_run).to include_recipe('yum')
        end

        it 'creates the user and groups' do
          expect(chef_run).to create_user('preview')
          expect(chef_run).to create_group('preview')
        end

        it 'prepares the preview service' do
          expect(chef_run).to create_cookbook_file('/etc/init.d/preview')
        end

        it 'places configuration' do
          expect(chef_run).to create_template('/etc/preview.conf')
        end

      end

      context "package install type on #{platform_name} #{platform_version}" do

        let(:chef_run) do
          ChefSpec::Runner.new(platform: platform_name, version: platform_version) do |node|
            node.set['preview']['install_type'] = 'package'
          end.converge('preview::app')
        end

        it 'installs the preview package' do
          expect(chef_run).to install_package('preview')
        end

      end

      context "archive install type on #{platform_name} #{platform_version}" do

        let(:chef_run) do
          ChefSpec::Runner.new(platform: platform_name, version: platform_version) do |node|
            node.set['preview']['install_type'] = 'archive'
          end.converge('preview::app')
        end

        it 'installs the preview archive and unpacks it' do

        end

      end

    end

  end

end
