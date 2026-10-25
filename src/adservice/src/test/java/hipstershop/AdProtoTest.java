package hipstershop;

import static org.junit.jupiter.api.Assertions.assertEquals;

import hipstershop.Demo.Ad;
import hipstershop.Demo.AdRequest;
import hipstershop.Demo.AdResponse;
import org.junit.jupiter.api.Test;

class AdProtoTest {

  @Test
  void adRequestKeepsContextKeys() {
    AdRequest request =
        AdRequest.newBuilder().addContextKeys("clothing").addContextKeys("cycling").build();

    assertEquals(2, request.getContextKeysCount());
    assertEquals("clothing", request.getContextKeys(0));
  }

  @Test
  void adKeepsTextAndRedirectUrl() {
    Ad ad = Ad.newBuilder().setRedirectUrl("/product/ABC").setText("Buy now").build();

    assertEquals("/product/ABC", ad.getRedirectUrl());
    assertEquals("Buy now", ad.getText());
  }

  @Test
  void adResponseHoldsAds() {
    Ad ad = Ad.newBuilder().setRedirectUrl("/product/ABC").setText("Buy now").build();
    AdResponse response = AdResponse.newBuilder().addAds(ad).build();

    assertEquals(1, response.getAdsCount());
    assertEquals("Buy now", response.getAds(0).getText());
  }

  @Test
  void adRequestSurvivesSerialization() throws Exception {
    AdRequest original = AdRequest.newBuilder().addContextKeys("kitchen").build();

    AdRequest parsed = AdRequest.parseFrom(original.toByteArray());

    assertEquals(original, parsed);
  }
}