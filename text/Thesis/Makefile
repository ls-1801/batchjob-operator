MDS=$(wildcard sections/*.md)
TEXS=$(MDS:sections/%.md=generated/%.tex)

generated/%.tex: sections/%.md
	pandoc $^ -f markdown -t latex -s -o $@ 

md: $(TEXS)

pdf: md
	pdflatex thesis.tex


clean:
	-rm $(TEXS)