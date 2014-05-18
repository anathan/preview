#
# Cookbook Name:: preview
# Recipe:: app
#
# Copyright (C) 2014 Nick Gerakines <nick@gerakines.net>
# 
# This project and its contents are open source under the MIT license.
#


include_recipe 'apt::default'
include_recipe 'yum::default'

require 'json'

user 'preview' do
  username 'preview'
  home '/home/preview'
  action :remove
  action :create
  supports ({ :manage_home => true })
end

group 'preview' do
  group_name 'preview'
  members 'preview'
  action :remove
  action :create
end

package 'unzip' do
	action :install
end

package 'curl' do
  action :install
end

case node[:preview][:install_type]
when 'package'
  package node[:preview][:package]

when 'archive'
  remote_file "#{Chef::Config[:file_cache_path]}/preview.zip" do
    source node[:preview][:archive_source]
  end

  bash 'extract_app' do
    cwd '/home/preview/'
    code <<-EOH
      unzip #{Chef::Config[:file_cache_path]}/preview.zip
      EOH
    not_if { ::File.exists?('/home/preview/preview') }
  end

  execute 'chown -R preview:preview /home/preview/'

  file '/home/preview/preview' do
    mode 00777
  end
end

template '/etc/preview.conf' do
  source 'preview.conf.erb'
  mode 0640
  group 'preview'
  owner 'preview'
  variables(:json => JSON.pretty_generate(node[:preview][:config].to_hash))
end

case node['platform_family']
when 'rhel'

  package 'ImageMagick'

  remote_file "#{Chef::Config[:file_cache_path]}/LibreOffice_4.2.4_Linux_x86-64_rpm.tar.gz" do
    source "http://download.documentfoundation.org/libreoffice/stable/4.2.4/rpm/x86_64/LibreOffice_4.2.4_Linux_x86-64_rpm.tar.gz"
  end

  directory '/opt/yum/libreoffice/' do
    owner 'root'
    group 'root'
    recursive true
    mode 00644
    action :create
  end

  package 'createrepo'

  bash 'unpack libreoffice' do
    cwd '/opt/yum/libreoffice/'
    code <<-EOH
      tar zxvf #{Chef::Config[:file_cache_path]}/LibreOffice_4.2.4_Linux_x86-64_rpm.tar.gz
      createrepo .
      EOH
  end

  yum_repository 'libreoffice-local' do
    description 'libreoffice-local'
    baseurl 'file:///opt/yum/libreoffice/'
    gpgcheck false
    enabled true
    action :create
  end

  yum_package 'libreoffice4.2-calc' do
    action :install
    flush_cache [:before]
  end

  %w{libreoffice4.2-impress libreoffice4.2-math libreoffice4.2-writer poppler-utils}.each do |pkg|
      yum_package pkg do
        action :install
      end
  end

  link '/usr/bin/soffice' do
    to '/opt/libreoffice4.2/program/soffice.bin'
  end

when 'debian', 'mac_os_x'
  package 'imagemagick'
end
