#!/usr/bin/env rake

require 'foodcritic'
require 'rake/testtask'

FoodCritic::Rake::LintTask.new do |t|
  t.options = { :fail_tags => ['any'] }
end

task :default => :foodcritic
