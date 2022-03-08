MDS=$(wildcard md/*/*.md)
TEXS=$(MDS:md/%.md=generated/%.tex)

generated/%.tex: md/%.md
	pandoc $^ -f markdown --natbib -t latex -s -o $@ 

debug: 
	echo $(TEXS)

md: $(TEXS)

pdf: md
	pdflatex thesis.tex


clean:
	-rm $(TEXS)