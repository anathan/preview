#
# Cookbook Name:: preview
# Recipe:: build
#
# Copyright (C) 2014 Nick Gerakines <nick@gerakines.net>
# 
# This project and its contents are open source under the MIT license.
#

include_recipe 'golang::default'

node.default['go']['packages'] = ['github.com/gpmgo/gopm']

include_recipe 'golang::packages'
