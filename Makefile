CXX      := g++
CXXFLAGS := -std=c++17 -Wall -Wextra -O2 -Iinclude
LDFLAGS  := -lssl -lcrypto

TARGET   := nopile
SRCS     := $(wildcard src/*.cpp)
OBJS     := $(SRCS:src/%.cpp=build/%.o)

PREFIX   := /usr
BINDIR   := $(PREFIX)/bin

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
	mkdir -p /var/lib/nopile/packages
	mkdir -p /var/lib/nopile/sources
	mkdir -p /var/cache/nopile
	@echo "nopile installed → $(BINDIR)/$(TARGET)"

uninstall:
	rm -f $(BINDIR)/$(TARGET)

clean:
	rm -rf build $(TARGET)
