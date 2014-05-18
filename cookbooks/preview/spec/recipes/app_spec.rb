require 'chefspec'
require 'chefspec/berkshelf'
ChefSpec::Coverage.start!

platforms = {
  # 'ubuntu' => ['12.04', '13.10'],
  'centos' => ['5.9', '6.5']
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

        it 'installs required packages' do
          expect(chef_run).to install_package('curl')
          expect(chef_run).to install_package('unzip')
        end

        it 'places configuration' do
          expect(chef_run).to create_template('/etc/preview.conf')
        end

        it 'installs required render agent packages' do
          expect(chef_run).to install_package('ImageMagick')
          expect(chef_run).to install_package('createrepo')
          expect(chef_run).to install_yum_package('libreoffice4.2-calc')
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
          expect(chef_run).to create_remote_file('/var/chef/cache/preview.zip')
          expect(chef_run).to run_bash('extract_app')
          expect(chef_run).to run_execute('chown -R preview:preview /home/preview/')
          expect(chef_run).to create_file('/home/preview/preview')
        end

      end

    end

  end

end
