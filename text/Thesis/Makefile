MDS=$(wildcard md/*/*.md)
TEXS=$(MDS:md/%.md=generated/%.tex)

generated/%.tex: md/%.md
	pandoc $^ -f markdown --natbib --highlight-style=tango --lua-filter=filters/short-captions.lua --filter pandoc-crossref -t latex -s -o $@ 

debug: 
	echo $(TEXS)

md: $(TEXS)

bib: md
	bibtex thesis

pdf: md
	pdflatex thesis.tex -interaction=nonstopmode


clean:
	-rm $(TEXS)