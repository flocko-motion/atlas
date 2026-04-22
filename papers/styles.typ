// Shared styling for all RankeDB papers.
//
// Usage at the top of each main.typ:
//   #import "../styles.typ": paper, admonition, abstract
//   #show: paper.with(
//     title: "...",
//     author: "Florian Noël",
//     byline: [Florian Noël \ 2026-04-09 --- draft --- CC-BY-4.0],
//     // or — for papers without a formal byline — set `tagline` instead:
//     // tagline: "A short italic note under the title.",
//   )
//
//   #abstract[ ... ]                  // renders an unnumbered "Abstract" heading
//   #admonition[ ... ]                // grey callout block (e.g. editorial notes, TODOs)
//   #admonition(fill: luma(225))[...] // lighter shade for nested callouts
//
//   = First section
//   ...

#let paper(
  title: "",
  subtitle: none,
  author: "",
  byline: none,
  tagline: none,
  body,
) = {
  let doc_title = if subtitle != none { title + ": " + subtitle } else { title }
  set document(title: doc_title, author: author)
  set page(numbering: "1")
  set par(justify: true)
  set heading(numbering: "1.")

  align(center)[
    #text(size: 18pt, weight: "bold")[#title]

    #if subtitle != none {
      v(0.3em)
      text(size: 13pt)[#subtitle]
    }

    #if tagline != none {
      v(0.5em)
      emph(tagline)
    }

    #if byline != none {
      v(0.5em)
      byline
    }
  ]

  v(1em)

  body
}

#let admonition(body, fill: luma(240)) = block(
  fill: fill,
  inset: 10pt,
  radius: 4pt,
  width: 100%,
  body,
)

#let abstract(body) = {
  heading(numbering: none, outlined: false, level: 2)[Abstract]
  body
}
