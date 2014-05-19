name             'preview_prod'
maintainer       'Nick Gerakines'
maintainer_email 'nick@gerakines.net'
license          'MIT'
description      'Installs/Configures preview_prod'
long_description IO.read(File.join(File.dirname(__FILE__), 'README.md'))
version          '0.2.2'

depends 'preview'

supports 'centos'

attribute 'preview_prod/node_id',
  :display_name => 'The id of the preview node.',
  :required => 'required',
  :type => 'string',
  :recipes => ['preview_prod::node']

attribute 'preview_prod/cassandra_hosts',
  :display_name => 'The cassandra hosts used by the preview node.',
  :required => 'required',
  :type => 'array',
  :recipes => ['preview_prod::node']

attribute 'preview_prod/edge_host',
  :display_name => 'The base url used to request assets from the cluster.',
  :required => 'required',
  :type => 'string',
  :recipes => ['preview_prod::node']

attribute 'preview_prod/s3Key',
  :display_name => 'The S3 key used to store generated assets.',
  :required => 'required',
  :type => 'string',
  :recipes => ['preview_prod::node']

attribute 'preview_prod/s3Secret',
  :display_name => 'The S3 secret key used to store generated assets.',
  :required => 'required',
  :type => 'string',
  :recipes => ['preview_prod::node']

attribute 'preview_prod/s3Host',
  :display_name => 'The S3 host used to store generated assets.',
  :required => 'required',
  :type => 'string',
  :recipes => ['preview_prod::node']

attribute 'preview_prod/s3Buckets',
  :display_name => 'The S3 buckets used to store generated assets.',
  :required => 'required',
  :type => 'array',
  :recipes => ['preview_prod::node']
