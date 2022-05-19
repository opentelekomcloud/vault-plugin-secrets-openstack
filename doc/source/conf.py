# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
# implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# -- General configuration ----------------------------------------------------

# Add any Sphinx extension module names here, as strings. They can be
# extensions coming with Sphinx (named 'sphinx.ext.*') or your custom ones.
extensions = [
    'myst_parser',
    'otcdocstheme',
]

# openstackdocstheme options
otcdocs_repo_name = 'opentelekomcloud/vault-plugin-secrets-openstack'
html_last_updated_fmt = '%Y-%m-%d %H:%M'
html_theme = 'otcdocs'

# The suffix of source filenames.
source_suffix = {
    '.rst': 'restructuredtext',
    '.txt': 'markdown',
    '.md': 'markdown',
}

# The master toctree document.
master_doc = 'index'

# General information about the project.
project = u'vault-plugin-secrets-openstack'
copyright = u'2022, Various members of the OpenTelekomCloud'

otcdocs_bug_tag = "docs"
# html_context allows us to pass arbitrary values into the html template
html_context = {}

# If true, '()' will be appended to :func: etc. cross-reference text.
add_function_parentheses = True

# If true, the current module name will be prepended to all description
# unit titles (such as .. function::).
add_module_names = True

# The name of the Pygments (syntax highlighting) style to use.
pygments_style = 'native'

autodoc_member_order = "bysource"

# Locations to exclude when looking for source files.
exclude_patterns = []

# -- Options for HTML output ----------------------------------------------

# Don't let openstackdocstheme insert TOCs automatically.
theme_include_auto_toc = False

# Output file base name for HTML help builder.
htmlhelp_basename = '%sdoc' % project

# Grouping the document tree into LaTeX files. List of tuples
# (source start file, target name, title, author, documentclass
# [howto/manual]).
latex_documents = [
    ('index',
     '%s.tex' % project,
     u'%s Documentation' % project,
     u'OpenTelekomCloud', 'manual'),
]

# Include both the class and __init__ docstrings when describing the class
autoclass_content = "both"
