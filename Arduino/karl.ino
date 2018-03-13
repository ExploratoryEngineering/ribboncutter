/**
 * Ribbon cutting device.
 */

#include <TelenorNBIOT.h>
#include <Adafruit_NeoPixel.h>

#define RX_PIN 10
#define TX_PIN 11
#define WIRE_PIN 2

#define STATE_INIT 0
#define STATE_WAITING 1
#define STATE_ACTIVE 2
int deviceState = STATE_INIT;

#define PIXEL_COUNT 20
#define PIXEL_PIN 7

TelenorNBIoT nbiot = TelenorNBIoT(RX_PIN, TX_PIN);
Adafruit_NeoPixel strip = Adafruit_NeoPixel(PIXEL_COUNT, PIXEL_PIN, NEO_GRB + NEO_KHZ800);

/**
 * Turn off all leds
 */
void allOff()
{
    strip.begin();
    for (uint16_t i = 0; i < strip.numPixels(); i++)
    {
        strip.setPixelColor(i, 0);
    }
    strip.show();
}

/**
 * Set all leds to red
 */
void allRed()
{
    strip.begin();
    for (uint16_t i = 0; i < strip.numPixels(); i++)
    {
        strip.setPixelColor(i, strip.Color(255, 0, 0));
    }
    strip.show();
}

/**
 * Go into error mode. Blink a series of blinks forever.
 */
void showError(uint8_t blinks)
{
    for (;;)
    {
        for (uint8_t i = 0; i < blinks; i++)
        {
            allRed();
            delay(500);
            allOff();
            delay(500);
        }
        delay(1000);
    }
}
void setup()
{
    pinMode(WIRE_PIN, INPUT_PULLUP);
    if (digitalRead(WIRE_PIN) == HIGH)
    {
        showError(3);
    }
    nbiot.begin();
    if (!nbiot.connect())
    {
        showError(1);
    }
}

int interval = 0;
const uint8_t bright[] = {1, 2, 3, 4, 8, 10, 12, 14, 16, 18, 20, 24, 28, 32, 36, 40, 44, 48, 52, 56, 60, 64, 72, 80, 88, 96, 128, 255};
int direction = 1;
int index = 0;

/**
 * Waiting pulse - adjust brightness of all leds up and down to signal waiting state
 */
void waitingPulse()
{
    strip.begin();
    for (uint16_t i = 0; i < strip.numPixels(); i++)
    {
        strip.setPixelColor(i, strip.Color(0, bright[index], 0));
    }
    strip.show();
    index += direction;
    if (index > strip.numPixels() || index < 0)
    {
        direction = -direction;
        index += direction;
    }
    delay(40);
}

/**
 * Send a completed message
 */
void triggerCompleteMessage()
{
    if (!nbiot.send("NTNU", 4))
    {
        showError(2);
    }
}

int pixelStart = 0;
int pixelNum = 0;

/**
 * Signal active by rotating a series of blue pixels around the top of the.... thing
 */
void activePulse()
{
    strip.begin();
    for (uint16_t i = 0; i < strip.numPixels(); i++)
    {
        if ((pixelNum + 5) % PIXEL_COUNT == i)
        {
            strip.setPixelColor(i, strip.Color(0, 0, 255));
        }
        else if ((pixelNum + 4) % PIXEL_COUNT == i)
        {
            strip.setPixelColor(i, strip.Color(0, 0, 255));
        }
        else if ((pixelNum + 3) % PIXEL_COUNT == i)
        {
            strip.setPixelColor(i, strip.Color(0, 0, 64));
        }
        else if ((pixelNum + 2) % PIXEL_COUNT == i)
        {
            strip.setPixelColor(i, strip.Color(0, 0, 64));
        }
        else if ((pixelNum + 1) % PIXEL_COUNT == i)
        {
            strip.setPixelColor(i, strip.Color(0, 0, 64));
        }
        else if (pixelNum == i)
        {
            strip.setPixelColor(i, strip.Color(0, 0, 32));
        }
        else
        {
            strip.setPixelColor(i, strip.Color(0, 0, 0));
        }
    }
    strip.show();
    pixelNum++;
    if (pixelNum > PIXEL_COUNT)
    {
        pixelNum = 0;
    }
    delay(40);
}

/**
 * Signal send procedure by changing all the lights to purple
 */
void sendPulse()
{
    strip.begin();
    for (uint16_t i = 0; i < strip.numPixels(); i++)
    {
        strip.setPixelColor(i, strip.Color(128, 0, 128));
    }
    strip.show();
}

int counter = 0;

#define LOOP_DELAY 20
#define COUNTER_SEND_INTERVAL ((1000 / LOOP_DELAY) * 10)

void loop()
{
    int wireState = digitalRead(WIRE_PIN);

    if (wireState == HIGH && deviceState != STATE_ACTIVE)
    {
        deviceState = STATE_ACTIVE;
        sendPulse();
        triggerCompleteMessage();
    }
    if (wireState == LOW && deviceState != STATE_WAITING)
    {
        deviceState = STATE_WAITING;
    }

    pixelStart++;
    if (pixelStart >= PIXEL_COUNT)
    {
        pixelStart = 0;
    }

    switch (deviceState)
    {
    case STATE_WAITING:
        waitingPulse();
        break;

    case STATE_ACTIVE:
        counter++;
        if (counter >= COUNTER_SEND_INTERVAL)
        {
            triggerCompleteMessage();
            counter = 0;
        }
        activePulse();
        break;

    default:
        allOff();
        break;
    }
}
