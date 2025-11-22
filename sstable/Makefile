CXX = g++
CXXFLAGS = -std=c++11 -Wall -Wextra -O2 -fPIC
SRCDIR = .
OBJDIR = .
TARGET = libsstable.a

SOURCES = $(SRCDIR)/sstable.cpp
OBJECTS = $(OBJDIR)/sstable.o

.PHONY: all clean

all: $(TARGET)

$(TARGET): $(OBJECTS)
	ar rcs $(TARGET) $(OBJECTS)

$(OBJDIR)/sstable.o: $(SRCDIR)/sstable.cpp $(SRCDIR)/sstable.h
	$(CXX) $(CXXFLAGS) -c $< -o $@

clean:
	rm -f $(OBJECTS) $(TARGET)

