CXX      := g++
CXXFLAGS := -std=c++17 -Wall -Wextra -O2 -Iinclude
LDFLAGS  :=

TARGET   := nopile
SRCS     := $(wildcard src/*.cpp)
OBJS     := $(SRCS:src/%.cpp=build/%.o)

PREFIX   := /usr
BINDIR   := $(PREFIX)/bin
MANDIR   := $(PREFIX)/share/man/man1

.PHONY: all clean install uninstall

all: $(TARGET)

$(TARGET): $(OBJS)
	$(CXX) $(CXXFLAGS) -o $@ $^ $(LDFLAGS)

build/%.o: src/%.cpp | build
	$(CXX) $(CXXFLAGS) -c -o $@ $<

build:
	mkdir -p build

install: $(TARGET)
	install -Dm755 $(TARGET) $(DESTDIR)$(BINDIR)/$(TARGET)
	@[ -f man/nopile.1 ] && install -Dm644 man/nopile.1 $(DESTDIR)$(MANDIR)/nopile.1 || true
	mkdir -p $(DESTDIR)/var/lib/nopile/packages
	mkdir -p $(DESTDIR)/var/cache/nopile
	@echo "nopile installed in $(BINDIR)/$(TARGET)"

uninstall:
	rm -f $(BINDIR)/$(TARGET)
	rm -f $(MANDIR)/nopile.1

clean:
	rm -rf build $(TARGET)
