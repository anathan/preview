#
# Cookbook Name:: preview
# Recipe:: deploy
#
# Copyright (C) 2014 Nick Gerakines <nick@gerakines.net>
# 
# This project and its contents are open source under the MIT license.
#

cookbook_file '/etc/init.d/preview' do
  source 'preview'
  mode 00777
  owner 'root'
  group 'root'
end

service 'preview' do
  provider Chef::Provider::Service::Init
  action [:start]
end
