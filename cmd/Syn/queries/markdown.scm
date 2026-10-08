(atx_heading (inline) @text.title)
(setext_heading (paragraph) @text.title)
[ (atx_h1_marker) (atx_h2_marker) (atx_h3_marker) (atx_h4_marker) (atx_h5_marker) (atx_h6_marker)
  (setext_h1_underline) (setext_h2_underline) ] @text.title
[ (link_title) (indented_code_block) ] @text.literal
; Fenced blocks: the fences and the info string here; the content is
; highlighted with the language the info string names (fence.go).
(fenced_code_block (fenced_code_block_delimiter) @text.literal)
(fenced_code_block (info_string) @text.literal)
[ (link_destination) ] @text.uri
[ (link_label) ] @text.reference
[ (list_marker_plus) (list_marker_minus) (list_marker_star) (list_marker_dot) (list_marker_parenthesis)
  (thematic_break) (block_quote_marker) ] @punctuation.special
