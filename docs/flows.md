# Flowcharts

#### General Relayer Flow 
The diagram on the left outlines the general flow of starting the relayer. 

#### Chain Initialization
The diagram in the middle details the steps taken by the relayer to initialize all of the chains setup in the configuration. 

#### Listening, Broadcasting, and Flushing
The diagram on the right displays how the listeners, broadcasters, and the chain's corresponding flushing mechanism are initialized.

[![](https://mermaid.ink/img/pako:eNp1Vg1vozgQ_SsW0kq7p7bqV7ZppN0ToaRBpZAFssndcUIuOI2vxEZAuu1W_e83xkAMSStVis2b8fPMewNvWswToo20T5_QDVlRRtAk5b8KRBnyyDPJC4LcPCE5Kjgq1-QV4SwjOEc2WZXo-Dvy6OO6DNkKguI1zktkeyELWbCMbMsPTMf0Is-dB5Zj_hNqwQuyaVESBvk8vi3huFD7N2SWHznu2DbfQs0qkMMfUoKMNabsz1B7D5kTGZ6pB2Y0tl3jrk0MCf1SHDlOefx0OHMbunC9O9Pz25gl-rEl-Sta8PxpF1NUQWYT5M_HvuFZs8BynTbSDKaSHPK3D0Wc06yknHVOhQSu48_vTeXuMtjgrNhuCIJKdANs172LxrpxB9AGNIUbceD4i5ZrOM7m_OkBw009zB7bwk3suT-VhZuk22KNTIahfoksnR_oXiAxe1wkvENjaQW7vk1sdwFg84WWu-IKdVTYEDg-POY4WyO12XWQ2uk2BMHfAWEcg4waBUhQs0LHx-iVFEggPhBB91ndZVSnrO69n5LxCnCwzc2Tbv_q3bZJhw-oVirnA9WXmfbqvJ-jIfkBlLBE9ECeYEx1y6m0nYFO5lmtz46orcD0xGVbrFWSHJe10SrDg-5W9LFC35pB5JjLAHC3pEQOeSklUOZygJRhW6YTVJkYLSlO6W9IllLCSnli4EGloqlp3U6DVnRTIgYGCnIQ8v4UkOBo4s6dG6npGq8_Y5oKXUtZVwTGnqvfGDqUxuuRqG4_zjlOYgwizKW4nZtIn80EE_rIcIr0LEN-yTNJdqnOlT8O6neh2zbU5d4MPMvYjZIFTlOo0T0pcxoX_SvZQFBeJQUuKN6NtQncAoThzywnmgteE8ooOLJuYt9lNa5uYGO0TseLrtdgptt0A-6F8QHHi-FecvTfFmhcoJgzRmIxugrEczHbUUKLLMVi3qQpeiCIr1YnMpMqM6n9jpqqrUYyMqJZSbSil2pDlUbjJbX3rR3UTdUVspcfwhoD9nVSB_R2JaVld6Z0m92QFN1sTxULhVS3AMrzhk234Qd87Jm2_pfpff7cSMsj0A-h3y9fpIAVhMmS7nMbLiQG18S6BWHYoH7V0cDnp25bDUJK8icYpkGpLwx9ZrXyht_IJ_lzz60S2PdCY4JDAR39CpN1p1UrXWDqW7eObkdmNQVqu4J04coqSznto5nr2i0BMO4s5zEpCt57qy90SyjEi6Dj95aj1y_1BQaDrAAckHxDGRaO6FuvrnrjubrqXbN1Oli1W-lHIx-1A61M1M2exut0H0K7LxrolMpFNG73UBVytxUNuV3R2_N2WwqxveJ_AG_I7V1kL0FtuP3-HI6vPRMyGHCGHGPyqzVkaqfQt2_fvh8am_JB_6tFO9I2IAFME_gcfhMnhRpMxQ2odwQ_E5w_hVrI3gGHtyX3X1msjcp8S460bZbAm_SGYpDLRhutcFrAbobZ35xvGhAstdGb9qKNzi4HJ9fDs8HXs-H52fD04kh7hc2LwcnF2elX8T-4PIfl-5H2u4o_PbkaDi4H15eD4fD86vrq_PJIIwmFb8N7-fFefcO__w8ccapX?type=png)](https://mermaid.live/edit#pako:eNp1Vg1vozgQ_SsW0kq7p7bqV7ZppN0ToaRBpZAFssndcUIuOI2vxEZAuu1W_e83xkAMSStVis2b8fPMewNvWswToo20T5_QDVlRRtAk5b8KRBnyyDPJC4LcPCE5Kjgq1-QV4SwjOEc2WZXo-Dvy6OO6DNkKguI1zktkeyELWbCMbMsPTMf0Is-dB5Zj_hNqwQuyaVESBvk8vi3huFD7N2SWHznu2DbfQs0qkMMfUoKMNabsz1B7D5kTGZ6pB2Y0tl3jrk0MCf1SHDlOefx0OHMbunC9O9Pz25gl-rEl-Sta8PxpF1NUQWYT5M_HvuFZs8BynTbSDKaSHPK3D0Wc06yknHVOhQSu48_vTeXuMtjgrNhuCIJKdANs172LxrpxB9AGNIUbceD4i5ZrOM7m_OkBw009zB7bwk3suT-VhZuk22KNTIahfoksnR_oXiAxe1wkvENjaQW7vk1sdwFg84WWu-IKdVTYEDg-POY4WyO12XWQ2uk2BMHfAWEcg4waBUhQs0LHx-iVFEggPhBB91ndZVSnrO69n5LxCnCwzc2Tbv_q3bZJhw-oVirnA9WXmfbqvJ-jIfkBlLBE9ECeYEx1y6m0nYFO5lmtz46orcD0xGVbrFWSHJe10SrDg-5W9LFC35pB5JjLAHC3pEQOeSklUOZygJRhW6YTVJkYLSlO6W9IllLCSnli4EGloqlp3U6DVnRTIgYGCnIQ8v4UkOBo4s6dG6npGq8_Y5oKXUtZVwTGnqvfGDqUxuuRqG4_zjlOYgwizKW4nZtIn80EE_rIcIr0LEN-yTNJdqnOlT8O6neh2zbU5d4MPMvYjZIFTlOo0T0pcxoX_SvZQFBeJQUuKN6NtQncAoThzywnmgteE8ooOLJuYt9lNa5uYGO0TseLrtdgptt0A-6F8QHHi-FecvTfFmhcoJgzRmIxugrEczHbUUKLLMVi3qQpeiCIr1YnMpMqM6n9jpqqrUYyMqJZSbSil2pDlUbjJbX3rR3UTdUVspcfwhoD9nVSB_R2JaVld6Z0m92QFN1sTxULhVS3AMrzhk234Qd87Jm2_pfpff7cSMsj0A-h3y9fpIAVhMmS7nMbLiQG18S6BWHYoH7V0cDnp25bDUJK8icYpkGpLwx9ZrXyht_IJ_lzz60S2PdCY4JDAR39CpN1p1UrXWDqW7eObkdmNQVqu4J04coqSznto5nr2i0BMO4s5zEpCt57qy90SyjEi6Dj95aj1y_1BQaDrAAckHxDGRaO6FuvrnrjubrqXbN1Oli1W-lHIx-1A61M1M2exut0H0K7LxrolMpFNG73UBVytxUNuV3R2_N2WwqxveJ_AG_I7V1kL0FtuP3-HI6vPRMyGHCGHGPyqzVkaqfQt2_fvh8am_JB_6tFO9I2IAFME_gcfhMnhRpMxQ2odwQ_E5w_hVrI3gGHtyX3X1msjcp8S460bZbAm_SGYpDLRhutcFrAbobZ35xvGhAstdGb9qKNzi4HJ9fDs8HXs-H52fD04kh7hc2LwcnF2elX8T-4PIfl-5H2u4o_PbkaDi4H15eD4fD86vrq_PJIIwmFb8N7-fFefcO__w8ccapX)

#### For Editors
Please visit the embedded link in the markdown to modify the flowchart in the mermaid live editor. After changes are made, please update the image link in the markdown.