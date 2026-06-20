#import "@preview/fletcher:0.5.8" as fletcher: diagram, node, edge

#import fletcher.shapes: brace, diamond

#set page(width: 260mm, height: 200mm)

#let debug = false 
#set align(center + horizon)

#let rect_stroke = 1pt
#let rect_stroke_thin = 0.1pt
#let corner_radius = 5pt
#let def_outset = 5pt

#let col_flip  = orange 
#let col_ui = purple
#let col_backend = red 
#let col_render = blue
#let col_res = black 

#let img_pix = rect(stroke: none, inset: 0pt,image("./imgs/ui-pixl.png", height: 50pt), height: 50pt, width: 120pt)
#let img_ctl = rect(stroke: none, inset: 0pt,image("./imgs/ui-ctl.png", height: 50pt), height: 50pt, width: 120pt)
#let img_text = rect(stroke: none, inset: 0pt,image("./imgs/ui-text.png", height: 50pt), height: 50pt, width: 120pt)
#let img_web = rect(stroke: none, inset: 0pt, image("./imgs/ui-web.png", height: 50pt), height: 50pt, width: 120pt)

//#let edge_export = stroke(thickness: 1pt, paint: col_export)
#let edge_neutral = stroke(thickness: 1pt, paint: black)
//#let edge_import = stroke(thickness: 1pt, paint: col_import)


#let node_res(pos, content, name: link) = node(pos, content, stroke: (thickness: rect_stroke, paint: col_res), corner-radius: corner_radius, name: name, outset: def_outset)

#let node_source(pos, content, name: link) = node(pos,text(col_backend)[#content], stroke: (thickness: rect_stroke, paint: col_backend), corner-radius: corner_radius, name: name, outset: def_outset)

#let node_consumer(pos, content, name: link) = node(pos,text(col_render)[#content], stroke: (thickness: rect_stroke, paint: col_render), corner-radius: corner_radius, name: name, outset: def_outset)

#diagram(
  debug: debug,
node(enclose: ((0.5, 1),<title>, (7, 1)), corner-radius: corner_radius, name: <flipctl>, stroke: (thickness: rect_stroke, paint: col_flip)),

node((3, 1), [#text(col_flip)[FlipCtl]\ runs as deamon \ might access different backend application \ allows access from web, SSH,…], stroke: none, name: <title>),

node(enclose: (<backend>, <flipctl>, <flipper_link>), stroke: (thickness: rect_stroke, paint: black), corner-radius: corner_radius, name: <cc>),


node(enclose: ((-1, 1.5), <backend_sys>, <backend_apps>,<backend_link>), align(top, text(col_backend)[Backend]), stroke: (thickness: rect_stroke, paint: col_backend), name: <backend>, outset: def_outset, inset: 8pt),

node_source((-1, 2), [System \ Applications], name: <backend_sys>),
node_source((-1, 2.5), [Applications \ Wrappers], name: <backend_apps>),
node_source((-1, 3), [Link to \ other FlipCtl], name: <backend_link>),

node(enclose: ((1, 1.5), (1,7)),text(col_ui)[FlipperUI\ Interface], stroke: (thickness: rect_stroke, paint: col_ui), name: <flipperUI>, outset: def_outset),

node_consumer((3, 2),[Pixel Renderer],  name: <render_pixel>),
node_consumer((3, 4),[Text UI Renderer], name: <render_text>),
node_consumer((3, 5),[Web Renderer], name: <render_web>),
node_consumer((3, 6),[Event Listener], name: <event_listener>),
node_consumer((3, 7),[FlipperLink], name: <flipper_link>),

node_res((5,2), img_pix, name: <res_pix>), 
node_res((5,3), img_ctl, name: <res_ctl>), 
node_res((5,4), img_text, name: <res_text>), 
node_res((5,5), img_web, name: <res_web>), 
node_res((5,7), [Remote FlipCtl], name: <res_rem>), 

edge((-1, 3), (1, 3), "->", shift: (3pt, 3pt), label: "defines"),
edge((1, 3), (-1, 3), "-->", shift: (3pt, 3pt), label: "backcall", label-side: left),

edge((1,2), <render_pixel>, "->", label: "rendered by"),
edge((1,4), <render_text>, "->", label: "rendered by"),
edge((1,5), <render_web>, "->", label: "rendered by"),
edge((1,6), <event_listener>, "->", label: "defines"),
edge((1,7), <flipper_link>, "->", shift: (3pt, 3pt), label: "passes data + ui calls", label-side: right),
edge(<flipper_link>, (1,7) , "-->", shift: (3pt, 3pt), label: "initiate backcall",  bend: -20deg),


edge(<render_pixel>, <res_pix>, "->", shift: (0pt, 0pt), label: "render pixel"),
edge(<render_pixel>, (3.5, 2), (3.5, 3), <res_ctl>, "->", shift: (3pt, 0pt), label: "render pixel", label-pos: 0.8),

edge(<render_text>, <res_text>, "->", shift: (0pt, 0pt), label: "render as text"),
edge(<render_web>, <res_web>, "->", shift: (0pt, 0pt), label: "render as HTML"),

edge(<render_web>, <res_web>, "->", shift: (0pt, 0pt), label: "render as HTML"),

edge(<res_web>,(5,6), <event_listener>, "-->", label: "triggers", label-pos: 0.6),
edge(<event_listener>, (1,6), "-->", label: "initiate backcall", bend: -30deg),

edge(<flipper_link>, <res_rem>, "<->", label: [SSH connection\ USB connection\ SPI], label-sep: -13pt),


edge((0.25, 1),(-1,1), <backend>, "-->", label: "initiate source", label-pos: 0.3),


)
