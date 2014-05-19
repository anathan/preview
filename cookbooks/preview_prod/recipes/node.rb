#
# Cookbook Name:: preview_prod
# Recipe:: node
#
# Copyright (C) 2014 Nick Gerakines
# 
# Permission is hereby granted, free of charge, to any person obtaining
# a copy of this software and associated documentation files (the
# "Software"), to deal in the Software without restriction, including
# without limitation the rights to use, copy, modify, merge, publish,
# distribute, sublicense, and/or sell copies of the Software, and to
# permit persons to whom the Software is furnished to do so, subject to
# the following conditions:
# 
# The above copyright notice and this permission notice shall be
# included in all copies or substantial portions of the Software.
# 
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
# MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
# NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
# LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
# OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
# WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
#


node.override[:preview][:config][:common][:nodeId] = normal[:preview_prod][:node_id]
node.override[:preview][:config][:storage][:engine] = 'cassandra'
node.override[:preview][:config][:storage][:cassandraKeyspace] = 'preview'
node.override[:preview][:config][:storage][:cassandraKeyspace] = normal[:preview_prod][:cassandra_hosts]
node.override[:preview][:config][:simpleApi][:edgeBaseUrl] = normal[:preview_prod][:edge_host]
node.override[:preview][:config][:uploader][:engine] = "s3"
node.override[:preview][:config][:uploader][:s3Key] = normal[:preview_prod][:s3Key]
node.override[:preview][:config][:uploader][:s3Secret] = normal[:preview_prod][:s3Secret]
node.override[:preview][:config][:uploader][:s3Host] = normal[:preview_prod][:s3Host]
node.override[:preview][:config][:uploader][:s3Buckets] = normal[:preview_prod][:s3Buckets]

include_recipe 'preview::default'
