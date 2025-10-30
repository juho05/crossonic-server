#include <stdlib.h>
#include <fileref.h>
#include <flacfile.h>
#include <flacpicture.h>
#include <mp4file.h>
#include <id3v2tag.h>
#include <tbytevector.h>
#include <vector>
#include <tbytevectorstream.h>
#include <tfile.h>
#include <tlist.h>
#include <tpropertymap.h>
#include <attachedpictureframe.h>
#include <string.h>
#include <typeinfo>
#include <apefile.h>
#include <apetag.h>
#include <id3v1tag.h>
#include <xiphcomment.h>
#include <mpegfile.h>

#include "audiotags.h"

static bool unicodeStrings = true;

class ByteVectorStreamWithName : public TagLib::ByteVectorStream
{
public:
  ByteVectorStreamWithName(const char *name, const TagLib::ByteVector &data) : TagLib::ByteVectorStream(data)
  {
    this->fileName = TagLib::FileName(name);
  }
  TagLib::FileName name() const
  {
    return this->fileName;
  }

private:
  TagLib::FileName fileName;
};

const TagLib::AudioProperties *audiotags_file_audioproperties(const TagLib_FileRefRef *file);

Metadata* read(const char *filename, int checkHasImage) {
    TagLib_FileRefRef* file = audiotags_file_new(filename);
    if (file == NULL) {
        return NULL;
    }


    TagMap* map = audiotags_file_properties(file);
    if (map == NULL) {
        audiotags_file_close(file);
        return NULL;
    }


    Metadata* metadata = (Metadata*)malloc(sizeof(Metadata));
    metadata->tags = map;

    const TagLib::AudioProperties* props = audiotags_file_audioproperties(file);
    metadata->lengthMs = props->lengthInMilliseconds();
    metadata->bitRate = props->bitrate();
    metadata->sampleRate = props->sampleRate();
    metadata->channels = props->channels();

    if (checkHasImage) {
        metadata->hasImage = audiotags_has_picture(file);
    }

    audiotags_file_close(file);

    return metadata;
}

void free_metadata(Metadata* metadata) {
    if (metadata == NULL) return;

    if (metadata->tags != NULL) {
        if (metadata->tags->tags != NULL) {
            for (int i = 0; i < metadata->tags->size; ++i) {
                free(metadata->tags->tags[i].key);
                free(metadata->tags->tags[i].value);
            }
            free(metadata->tags->tags);
        }
        free(metadata->tags);
    }

    free(metadata);
}

TagLib_FileRefRef *audiotags_file_new(const char *filename)
{
  TagLib::FileRef *fr = new TagLib::FileRef(filename);
  if (fr == NULL || fr->isNull() || !fr->file()->isValid() || fr->tag() == NULL)
  {
    if (fr)
    {
      delete fr;
      fr = NULL;
    }
    return NULL;
  }

  TagLib_FileRefRef *holder = new TagLib_FileRefRef();
  holder->fileRef = reinterpret_cast<void *>(fr);
  holder->ioStream = NULL;
  return holder;
}

void audiotags_file_close(TagLib_FileRefRef *fileRefRef)
{
  delete reinterpret_cast<TagLib::FileRef *>(fileRefRef->fileRef);
  if (fileRefRef->ioStream)
  {
    delete reinterpret_cast<TagLib::IOStream *>(fileRefRef->ioStream);
  }
  delete fileRefRef;
}

TagMap* process_tags(const TagLib::PropertyMap &tags)
{
  int count = 0;
  for (TagLib::PropertyMap::ConstIterator i = tags.begin(); i != tags.end(); ++i)
  {
    count += i->second.size();
  }

  TagMap* map = (TagMap*)malloc(sizeof(TagMap));
  map->size = count;
  map->tags = (KeyValue*)malloc(map->size * sizeof(KeyValue));

  int index = 0;
  for (TagLib::PropertyMap::ConstIterator i = tags.begin(); i != tags.end(); ++i)
  {
    for (TagLib::StringList::ConstIterator j = i->second.begin(); j != i->second.end(); ++j)
    {
      char *key = ::strdup(i->first.toCString(unicodeStrings));
      char *val = ::strdup((*j).toCString(unicodeStrings));
      map->tags[index].key = key;
      map->tags[index].value = val;
      ++index;
    }
  }

  return map;
}

TagMap* audiotags_file_properties(const TagLib_FileRefRef *fileRefRef)
{
  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);

  if (TagLib::MPEG::File *mpeg = dynamic_cast<TagLib::MPEG::File *>(fileRef->file()))
  {
    if (auto id3v2Tag = mpeg->ID3v2Tag(false))
    {
      return process_tags(id3v2Tag->properties());
    }
    else if (auto id3v1Tag = mpeg->ID3v1Tag(false))
    {
      return process_tags(id3v1Tag->properties());
    }
  }
  else
  {
    return process_tags(fileRef->file()->properties());
  }
  return NULL;
}

int write_tag(const char* filename, const char* key, const char* value) {
  TagLib_FileRefRef* fileRefRef = audiotags_file_new(filename);
  if (fileRefRef == NULL) {
      return 0;
  }
  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);

  TagLib::String keyStr = key;
  TagLib::String valueStr = value;

  TagLib::PropertyMap tags = fileRef->file()->properties();
  if (tags.contains(keyStr)) {
    tags.replace(keyStr, valueStr);
  } else {
    tags.insert(keyStr, valueStr);
  }
  fileRef->file()->setProperties(tags);
  fileRef->file()->save();


  audiotags_file_close(fileRefRef);
  return 1;
}

int remove_crossonic_id(const char* filename, const char* instanceId) {
  TagLib_FileRefRef* fileRefRef = audiotags_file_new(filename);
  if (fileRefRef == NULL) {
      return 0;
  }
  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);

  TagLib::String keyStr = "CROSSONIC_ID_";
  if (strlen(instanceId) > 0) {
    keyStr += instanceId;
  }

  TagLib::PropertyMap tags = fileRef->file()->properties();
  if (strlen(instanceId) > 0) {
    if (tags.contains(keyStr)) {
      tags.erase(keyStr);
    }
  } else {
      std::vector<TagLib::String> toDelete;
      for (TagLib::PropertyMap::ConstIterator i = tags.begin(); i != tags.end(); ++i)
      {
        if (i->first.startsWith("CROSSONIC_ID_")) {
          toDelete.push_back(i->first);
        }
      }
      for (TagLib::String key : toDelete) {
        tags.erase(key);
      }
  }
  fileRef->file()->setProperties(tags);
  fileRef->file()->save();


  audiotags_file_close(fileRefRef);
  return 1;
}

const TagLib::AudioProperties *audiotags_file_audioproperties(const TagLib_FileRefRef *fileRefRef)
{
  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);
  return fileRef->file()->audioProperties();
}

bool audiotags_has_picture(TagLib_FileRefRef *fileRefRef)
{
  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);

  TagLib::ByteVector imageData;
  if (TagLib::FLAC::File *flac = dynamic_cast<TagLib::FLAC::File *>(fileRef->file()))
  {
    auto pictures = flac->pictureList();
    for (auto it = pictures.begin(); it != pictures.end(); ++it)
    {
      if ((*it)->type() == TagLib::FLAC::Picture::Type::FrontCover)
      {
        return true;
      }
    }
  }
  else if (TagLib::APE::File *ape = dynamic_cast<TagLib::APE::File *>(fileRef->file()))
  {
    if (auto apeTag = ape->APETag(false))
    {
      printf("\nape tag !!\n");
    }
  }
  else if (TagLib::MPEG::File *mpeg = dynamic_cast<TagLib::MPEG::File *>(fileRef->file()))
  {
    if (auto id3v2Tag = mpeg->ID3v2Tag(false))
    {
      auto frames = id3v2Tag->frameList();
      for (auto it = frames.begin(); it != frames.end(); ++it)
      {
        if (auto *pFrame = dynamic_cast<TagLib::ID3v2::AttachedPictureFrame *>(*it))
        {
          return true;
        }
      }
    }
  }
  else
  {
    auto tags = fileRef->file()->tag();
    if (auto mp4Tag = dynamic_cast<TagLib::MP4::Tag *>(tags))
    {
      TagLib::MP4::CoverArtList covers = mp4Tag->item("covr").toCoverArtList();
      if (!covers.isEmpty())
      {
        return true;
      }
    }
    else if (auto oggTag = dynamic_cast<TagLib::Ogg::XiphComment *>(tags))
    {
      auto pictures = oggTag->pictureList();
      for (auto it = pictures.begin(); it != pictures.end(); ++it)
      {
        if ((*it)->type() == TagLib::FLAC::Picture::Type::FrontCover)
        {
          return true;
        }
      }
    }
    else if (auto id3Tag = dynamic_cast<TagLib::ID3v2::Tag *>(tags))
    {
      auto frames = id3Tag->frameList();
      for (auto it = frames.begin(); it != frames.end(); ++it)
      {
        if (auto *pFrame = dynamic_cast<TagLib::ID3v2::AttachedPictureFrame *>(*it))
        {
          return true;
        }
      }
    }
  }
  return false;
}

void read_picture(const char* filename, int id)
{
  TagLib_FileRefRef* fileRefRef = audiotags_file_new(filename);
  if (fileRefRef == NULL) {
      return;
  }

  const TagLib::FileRef *fileRef = reinterpret_cast<const TagLib::FileRef *>(fileRefRef->fileRef);

  TagLib::ByteVector imageData;
  if (TagLib::FLAC::File *flac = dynamic_cast<TagLib::FLAC::File *>(fileRef->file()))
  {
    auto pictures = flac->pictureList();
    for (auto it = pictures.begin(); it != pictures.end(); ++it)
    {
      if ((*it)->type() == TagLib::FLAC::Picture::Type::FrontCover)
      {
        imageData = (*it)->data();
        break;
      }
    }
  }
  else if (TagLib::APE::File *ape = dynamic_cast<TagLib::APE::File *>(fileRef->file()))
  {
    if (auto apeTag = ape->APETag(false))
    {
      printf("\nape tag !!\n");
    }
  }
  else if (TagLib::MPEG::File *mpeg = dynamic_cast<TagLib::MPEG::File *>(fileRef->file()))
  {
    if (auto id3v2Tag = mpeg->ID3v2Tag(false))
    {
      auto frames = id3v2Tag->frameList();
      for (auto it = frames.begin(); it != frames.end(); ++it)
      {
        if (auto *pFrame = dynamic_cast<TagLib::ID3v2::AttachedPictureFrame *>(*it))
        {
          imageData = pFrame->picture();
          break;
        }
      }
    }
  }
  else
  {
    auto tags = fileRef->file()->tag();
    if (auto mp4Tag = dynamic_cast<TagLib::MP4::Tag *>(tags))
    {
      TagLib::MP4::CoverArtList covers = mp4Tag->item("covr").toCoverArtList();
      if (!covers.isEmpty())
      {
        imageData = covers.front().data();
      }
    }
    else if (auto oggTag = dynamic_cast<TagLib::Ogg::XiphComment *>(tags))
    {
      auto pictures = oggTag->pictureList();
      for (auto it = pictures.begin(); it != pictures.end(); ++it)
      {
        if ((*it)->type() == TagLib::FLAC::Picture::Type::FrontCover)
        {
          imageData = (*it)->data();
          break;
        }
      }
    }
    else if (auto id3Tag = dynamic_cast<TagLib::ID3v2::Tag *>(tags))
    {
      auto frames = id3Tag->frameList();
      for (auto it = frames.begin(); it != frames.end(); ++it)
      {
        if (auto *pFrame = dynamic_cast<TagLib::ID3v2::AttachedPictureFrame *>(*it))
        {
          imageData = pFrame->picture();
          break;
        }
      }
    }
  }
  if (!imageData.isEmpty())
  {
    goPutImage(id, imageData.data(), imageData.size());
  }
  audiotags_file_close(fileRefRef);
}