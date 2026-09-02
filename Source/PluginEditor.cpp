#include "PluginProcessor.h"
#include "PluginEditor.h"
#include <BinaryData.h>

VortexFocusAudioProcessorEditor::VortexFocusAudioProcessorEditor (VortexFocusAudioProcessor& p)
    : AudioProcessorEditor (&p), audioProcessor (p), webBrowser (
       #if JUCE_WEB_BROWSER_RESOURCE_PROVIDER_AVAILABLE
        juce::WebBrowserComponent::Options()
          .withResourceProvider([this](const juce::String& url) {
              if (url == "/" || url == "/index.html") {
                  auto* htmlData = BinaryData::index_html;
                  auto htmlSize = BinaryData::index_htmlSize;
                  return std::optional<juce::WebBrowserComponent::Resource>({
                      std::vector<std::byte>((const std::byte*)htmlData, (const std::byte*)htmlData + htmlSize),
                      "text/html"
                  });
              }
              return std::optional<juce::WebBrowserComponent::Resource>();
             })
         #else
          juce::WebBrowserComponent::Options()
         #endif
     )
{
    addAndMakeVisible(webBrowser);
    setSize(920, 820);

   #if JUCE_WEB_BROWSER_RESOURCE_PROVIDER_AVAILABLE
    webBrowser.goToURL (juce::WebBrowserComponent::getResourceProviderRoot());
   #else
    webBrowser.goToURL("data:text/html," + juce::URL::addEscapeChars(
        juce::String::fromUTF8(BinaryData::index_html, BinaryData::index_htmlSize), true));
   #endif
}

VortexFocusAudioProcessorEditor::~VortexFocusAudioProcessorEditor()
{
}

void VortexFocusAudioProcessorEditor::paint (juce::Graphics& g)
{
    g.fillAll (getLookAndFeel().findColour (juce::ResizableWindow::backgroundColourId));
}

void VortexFocusAudioProcessorEditor::resized()
{
    webBrowser.setBounds(getLocalBounds());
}